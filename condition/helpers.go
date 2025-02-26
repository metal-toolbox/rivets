package condition

import (
	"encoding/json"
	"net/url"
)

func (u *ConfigURL) UnmarshalJSON(data []byte) error {
	var rawURL string
	if err := json.Unmarshal(data, &rawURL); err != nil {
		return err
	}

	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return err
	}

	*u = ConfigURL(*parsedURL)
	return nil
}
