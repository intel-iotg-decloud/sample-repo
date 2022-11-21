// SPDX-FileCopyrightText: (C) 2022 Intel Corporation
// SPDX-License-Identifier: LicenseRef-Intel

package custom

// schema.go contains go struct that can be used to read or write
// the SRA config.json file.

type boolString string // "True" or "False"

type modelDefinition struct {
	Name        string `json:"name"`
	Description string `json:"modelDescription,omitempty"`
	Size        string `json:"modelSize,omitempty"`
}

type modelList []modelDefinition

type sourceDefinition struct {
	Name   string `json:"sourceName"`
	Source string `json:"source"`
	Type   string `json:"sourceType"`
}

type sourceRef struct {
	Name        string `json:"sourceName"`
	StreamCount uint8  `json:"streamCount"`
}

type nodeDef struct {
	Name      string `json:"nodeName"`
	Model     string `json:"model"`
	Precision string `json:"precision"`
	Device    string `json:"device"`
}

type pipelineDefinition struct {
	Name    string      `json:"pipelineName"`
	Info    string      `json:"info"`
	Enable  boolString  `json:"enable"`
	Sources []sourceRef `json:"sources"`
	Nodes   []nodeDef   `json:"nodeList"`
}

type sraConfig struct {
	Version   float32              `json:"version"`
	Models    map[string]modelList `json:"modelList"`
	Sources   []sourceDefinition   `json:"sourceList"`
	Pipelines []pipelineDefinition `json:"pipelineList"`
}
