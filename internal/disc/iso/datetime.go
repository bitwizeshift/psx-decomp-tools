package iso

import "time"

// decodeRecordingTime decodes the 7-byte timestamp ISO 9660 stores in a
// directory record (year since 1900, month, day, hour, minute, second, and a
// signed GMT offset in 15-minute units) from b into a [time.Time]. A timestamp
// whose every field is zero is reported as the zero [time.Time].
func decodeRecordingTime(b []byte) time.Time {
	if isZero(b[:7]) {
		return time.Time{}
	}
	offset := int(int8(b[6])) * 15 * 60
	return time.Date(
		1900+int(b[0]),
		time.Month(b[1]),
		int(b[2]),
		int(b[3]),
		int(b[4]),
		int(b[5]),
		0,
		time.FixedZone("", offset),
	)
}

// decodeVolumeTime decodes the 17-byte timestamp ISO 9660 stores in a volume
// descriptor (a "YYYYMMDDHHMMSShh" ASCII string followed by a signed GMT offset
// in 15-minute units) from b into a [time.Time]. An unset timestamp, whose year
// reads as zero, is reported as the zero [time.Time].
func decodeVolumeTime(b []byte) time.Time {
	year := atoi(b[0:4])
	if year == 0 {
		return time.Time{}
	}
	offset := int(int8(b[16])) * 15 * 60
	return time.Date(
		year,
		time.Month(atoi(b[4:6])),
		atoi(b[6:8]),
		atoi(b[8:10]),
		atoi(b[10:12]),
		atoi(b[12:14]),
		atoi(b[14:16])*10*int(time.Millisecond),
		time.FixedZone("", offset),
	)
}

// isZero reports whether every byte of b is zero.
func isZero(b []byte) bool {
	for _, v := range b {
		if v != 0 {
			return false
		}
	}
	return true
}

// atoi returns the base-10 value of the ASCII digits in b, which ISO 9660
// guarantees for the numeric fields of a volume timestamp.
func atoi(b []byte) int {
	n := 0
	for _, v := range b {
		n = n*10 + int(v-'0')
	}
	return n
}
