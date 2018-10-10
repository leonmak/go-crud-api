package utils

import "fmt"

func MakePointString(lat float64, lng float64) (pointString string) {
	return fmt.Sprintf("ST_MakePoint(%.6f,%.6f)", lat, lng)
}
