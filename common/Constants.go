package common

import "path/filepath"

const PythonFilename = "python"
const PayloadFilename = "payload"
const IntegrityFilename = "integrity_hashes"
const WheelsFilename = "wheels"
const HashesFilename = "hashes"

const pipFilename = "pip.pyz"

// pure mode is an optional feature that can be enabled to ensure that the installer does not run the embedded files after
// they have been deposited on disk;
// it only extracts them and then sets up a batch file to run the extracted files. This is useful if you want to sign
// the installer and ensure that the signature is not associated with potentially tampered files (after extraction).
const PureMode = true

func GetConfigEmbedName() string {
	return "settings.json"
}

func GetPipName(extractDir string) string {
	return filepath.Join(extractDir, pipFilename)
}
