package common

import (
	"errors"
	"fmt"
)

type Condition struct {
	Header string `json:"header"`
	Value  string `json:"value"`
}

func (c Condition) Validate() error {
	if c.Header == "" || c.Value == "" {
		return errors.New("both header and value must be set in condition")
	}
	return nil
}

func ParseCondition(condStr string) (Condition, error) {
	var header, value string
	if condStr != "" {
		n, err := fmt.Sscanf(condStr, "%[^=]=%s", &header, &value)
		if err != nil || n != 2 {
			return Condition{}, errors.New("invalid condition format, expected header=value")
		}
		return Condition{
			Header: header,
			Value:  value,
		}, nil
	} else {
		return Condition{}, nil
	}

}

func (c Condition) String() string {
	if c.Header == "" && c.Value == "" {
		return "---"
	}
	return fmt.Sprintf("%s=%s", c.Header, c.Value)
}
