#!/bin/bash
# Flash env sensor nodes over OTA and verify they actually came back.
#
#   ./flash.sh vault              flash one node
#   ./flash.sh --all              flash every node in dhcpd-iot.conf
#   ./flash.sh --all --except a b flash everything but a and b
#
# Nodes are flashed one at a time and each is verified before the next starts,
# so a bad build stops the rollout instead of taking out the fleet.
set -euo pipefail
cd "$(dirname "$0")"

ESPHOME=./.venv/bin/esphome
[ -x "$ESPHOME" ] || { echo "FATAL: $ESPHOME missing -- python3 -m venv .venv && .venv/bin/pip install esphome==2025.10.2"; exit 1; }
[ -f secrets.go ] || { echo "FATAL: secrets.go missing"; exit 1; }
if grep -q '"FIXME"' secrets.go; then
    echo "FATAL: secrets.go still contains FIXME placeholders."
    echo "       Flashing with a wrong WiFi password strands the node off the"
    echo "       network and it can only be recovered over USB. Refusing."
    exit 1
fi

# Regenerate configs/ from template.yaml + secrets.go. This also rewrites the
# DNS and DHCP files, which is where the name->IP mapping below comes from.
echo "==> regenerating configs"
go run .

declare -A IP
# host <name> { hardware ethernet <mac>; fixed-address <ip>; }
while read -r _ name _ _ _ _ _ addr _; do
    IP[$name]="${addr%;}"
done < dhcpd-iot.conf

if [ "${1:-}" = "--all" ]; then
    shift
    skip=" "
    [ "${1:-}" = "--except" ] && { shift; skip=" $* "; }
    nodes=()
    for n in $(printf '%s\n' "${!IP[@]}" | sort); do
        [[ "$skip" == *" $n "* ]] || nodes+=("$n")
    done
else
    nodes=("$@")
fi
[ ${#nodes[@]} -gt 0 ] || { echo "FATAL: no nodes given"; exit 1; }

echo "==> ${#nodes[@]} node(s) to flash: ${nodes[*]}"
failed=()

for node in "${nodes[@]}"; do
    ip="${IP[$node]:-}"
    [ -n "$ip" ] || { echo "!! $node: no IP in dhcpd-iot.conf, skipping"; failed+=("$node(no-ip)"); continue; }

    echo
    echo "================ $node ($ip) ================"
    # `esphome upload` does NOT build -- it expects the binary to already
    # exist. Compile first, and treat a build failure as fatal for the whole
    # run rather than limping on to the next node with a stale image.
    if ! "$ESPHOME" compile "configs/$node.yaml"; then
        echo "!! $node: COMPILE FAILED -- aborting rollout"
        exit 1
    fi
    if ! "$ESPHOME" upload "configs/$node.yaml" --device "$ip"; then
        echo "!! $node: OTA upload FAILED"
        failed+=("$node(upload)")
        continue
    fi

    # The node reboots into the new image. Wait for /metrics, which only the
    # new firmware serves -- so this doubles as proof the new image is running.
    #
    # Wait for uptime >= 45s before believing anything. The scd4x defers its
    # first i2c transaction by 1s and only publishes on its 30s interval, so a
    # node scraped the instant it answers looks like it has a dead chip.
    echo "-- waiting for $node to serve /metrics and settle ..."
    ok=""
    for _ in $(seq 1 60); do
        if curl -sf --max-time 5 "http://$ip/metrics" -o /tmp/m.$$ 2>/dev/null && [ -s /tmp/m.$$ ]; then
            up=$(grep 'esphome_sensor_value.*id="uptime"' /tmp/m.$$ | grep -oE '[0-9.]+$' | cut -d. -f1)
            if [ -n "$up" ] && [ "$up" -ge 45 ]; then ok=1; break; fi
        fi
        sleep 5
    done
    if [ -z "$ok" ]; then
        echo "!! $node: never served /metrics after 5 minutes"
        failed+=("$node(no-metrics)")
        continue
    fi

    # Both chips must be present and not marked failed. This is the whole point
    # of the exercise, so a node that comes back with a dead chip counts as a
    # failure even though the flash itself worked.
    bad=$(grep '^esphome_sensor_failed' /tmp/m.$$ | grep -E 'name="(scd4x|sht4x)' | grep -v '} 0$' || true)
    n_chips=$(grep -c 'esphome_sensor_failed.*name="\(scd4x\|sht4x\)' /tmp/m.$$ || true)
    # Nodes built without an SCD4x expose only the three sht4x-side series, so
    # take the expectation from the config rather than assuming both chips.
    if grep -q 'platform: scd4x' "configs/$node.yaml"; then want_chips=5; else want_chips=3; fi
    if [ -n "$bad" ]; then
        echo "!! $node: chip reported failed after flash:"; echo "$bad"
        failed+=("$node(chip-failed)")
    elif [ "$n_chips" -lt "$want_chips" ]; then
        echo "!! $node: only $n_chips chip sensors present, expected $want_chips"
        failed+=("$node(missing-sensors)")
    elif stale=$(grep -E 'esphome_sensor_value.*id="(scd4x|sht4x)_seconds_since_reading"' /tmp/m.$$ \
                 | awk -F' ' '$NF+0 > 120 {print}' | grep . ); then
        # "not failed" is not the same as "publishing". Catch a chip that
        # initialised but has since gone quiet.
        echo "!! $node: chip initialised but is not publishing:"; echo "$stale"
        failed+=("$node(stale)")
    else
        echo "-- $node OK: $n_chips chip sensors, none failed"
        grep -E 'name="(uptime|reset_reason|wifi_signal)"' /tmp/m.$$ | sed 's/^/     /' || true
    fi
    rm -f /tmp/m.$$
done

echo
if [ ${#failed[@]} -eq 0 ]; then
    echo "==> all ${#nodes[@]} node(s) flashed and verified"
else
    echo "==> ${#failed[@]} of ${#nodes[@]} node(s) had problems: ${failed[*]}"
    exit 1
fi
