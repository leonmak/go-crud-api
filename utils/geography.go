package utils

import "fmt"

func MakePointString(lat interface{}, lng interface{}) (pointString string) {
	return fmt.Sprintf("ST_MakePoint(%.6f,%.6f)", lat, lng)
}
