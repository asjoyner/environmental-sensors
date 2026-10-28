#pragma once
// TUF-2000M clock offset, for the tufN_clock_offset sensors in a64.yaml.
//
// The meters keep their own calendar in REG0053-0055 (wire addresses 52-54),
// BCD, high byte first per register:
//   reg52 = minute|second   reg53 = day|hour   reg54 = year%100|month
// (the same layout the tufN_set_clock buttons write; verified 2026-10-28 by
// reading meter 2 back after a set). The buttons write local time from SNTP,
// so the offset compares against local time too.
#include <cmath>
#include <cstddef>
#include <cstdint>
#include "esphome/core/time.h"

static int tuf_bcd(uint8_t b) {
  return ((b >> 4) > 9 || (b & 0x0F) > 9) ? -1 : (b >> 4) * 10 + (b & 0x0F);
}

// Days since 1970-01-01 for a proleptic Gregorian date (Howard Hinnant).
static long tuf_days_from_civil(int y, int m, int d) {
  y -= m <= 2;
  const long era = (y >= 0 ? y : y - 399) / 400;
  const unsigned yoe = (unsigned) (y - era * 400);
  const unsigned doy = (153 * (m + (m > 2 ? -3 : 9)) + 2) / 5 + d - 1;
  const unsigned doe = yoe * 365 + yoe / 4 - yoe / 100 + doy;
  return era * 146097 + (long) doe - 719468;
}

// Seconds the meter's clock is ahead of `now` (negative = behind). NAN while
// SNTP isn't synced or when the registers aren't a valid BCD date; a meter
// that reset to its factory date (2022-09-22) is valid and reads years behind.
// Takes a pointer and length so it works with the lambda's `data` whether it
// is a std::vector (ESPHome 2025) or a std::span (2026.9+).
static float tuf_clock_offset(const uint8_t *data, size_t len, size_t off, esphome::ESPTime now) {
  if (!now.is_valid() || len < off + 6)
    return NAN;
  const int mi = tuf_bcd(data[off]), se = tuf_bcd(data[off + 1]);
  const int da = tuf_bcd(data[off + 2]), ho = tuf_bcd(data[off + 3]);
  const int yr = tuf_bcd(data[off + 4]), mo = tuf_bcd(data[off + 5]);
  if (mi < 0 || mi > 59 || se < 0 || se > 59 || ho < 0 || ho > 23 || da < 1 || da > 31 ||
      yr < 0 || mo < 1 || mo > 12)
    return NAN;
  const double meter = tuf_days_from_civil(2000 + yr, mo, da) * 86400.0 + ho * 3600 + mi * 60 + se;
  const double ref = tuf_days_from_civil(now.year, now.month, now.day_of_month) * 86400.0 +
                     now.hour * 3600 + now.minute * 60 + now.second;
  return (float) (meter - ref);
}
