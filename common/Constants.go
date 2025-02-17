package common

import "path/filepath"

const PythonFilename = "python"
const PayloadFilename = "payload"
const IntegrityFilename = "integrity_hashes"
const WheelsFilename = "wheels"
const HashesFilename = "hashes"

const pipFilename = "pip.pyz"

func GetConfigEmbedName() string {
	return "settings.json"
}

func GetPipName(extractDir string) string {
	return filepath.Join(extractDir, pipFilename)
}
