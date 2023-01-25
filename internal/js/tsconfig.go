package js

import (
	"os"

	"github.com/yosuke-furukawa/json5/encoding/json5"
)

type CompilerOptions struct {
	BaseUrl string              `json:"baseUrl,omitempty"`
	Paths   map[string][]string `json:"paths,omitempty"`
}

type TsConfig struct {
	CompilerOptions CompilerOptions `json:"compilerOptions,omitempty"`
}

func ParseTsConfig(path string) (TsConfig, error) {
	var tsConfig TsConfig
	data, err := os.ReadFile(path)
	if err != nil {
		return TsConfig{}, err
	}
	err = json5.Unmarshal(data, &tsConfig)
	return tsConfig, err
}
