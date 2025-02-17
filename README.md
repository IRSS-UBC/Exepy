**Exepy "Ex-eh-pie"**

A straightforward tool for packaging your Python projects into standalone Windows executable files. Inspired by PyInstaller, Exepy offers an alternative built with Golang.

**Why Exepy?**

* **Effortless Distribution:** Share your Python applications with anyone, even users without Python installed. Exepy handles all dependencies for you.
* **Offline Compatibility:** Your packaged executables can be configured to work without needing an active internet connection, as all required libraries can be bundled within.
* **Unrestricted Use:** Exepy's public domain license (the Unlicense) allows you to copy, modify, and distribute your packaged projects for both personal and commercial purposes without limitations.

**Getting Started**

1. **Download Exepy:** Obtain the [latest release](https://github.com/IRSS-UBC/Exepy/releases).
2. **Structure Your Project:**
   * **Script Folder:** Place all your Python code into a designated folder named "payload".
   * **Requirements:** List your project's dependencies in a `requirements.txt` file. If you haven't already, use `pip freeze > requirements.txt` to generate this file. Place it in the scripts directory.
   * **Configuration:** Fine-tune Exepy's behavior with a `settings.json` file (explained below).
3. **Build Your Executable:** Run the Exepy binary, and it will prepare your Python environment and bundle it with your scripts into a single, ready-to-distribute executable file.

**Customization (settings.json)**

Exepy offers flexibility through its `settings.json` file. Here's a breakdown of the options:


* **pythonDownloadURL:** The URL to download the Python interpreter.
* **pipDownloadURL:** The URL to download the pip package manager.
* **pythonDownloadFile:** The name of the Python interpreter download file.
* **pythonExtractDir:** The directory to extract the Python interpreter to.
* **scriptExtractDir:** The directory to extract the scripts to.
* **pthFile:** The name of the .pth file inside the Python distribution.
* **pythonInteriorZip:** The name of the interior zip file inside the Python distribution.
* **installerRequirements:** Additional requirements for the installer to bundle with the executable.
* **requirementsFile:** The name of the requirements file in the scripts dir.
* **scriptDir:** The directory containing the scripts.
* **setupScript:** The script to run before the main script. (optional)
* **mainScript:** The main script to run your Python program.
* **filesToCopyToRoot:** A list of files to copy to the root of the executable.
* **runAfterInstall:** Whether to run the main script after installation or to instruct users to run the corresponding run.bat file.


**Example Default Configuration:**
```json
{
  "pythonDownloadURL": "",
  "pipDownloadURL": "",
  "pythonDownloadFile": "python code-3.11.7-embed-amd64.zip",
  "pythonExtractDir": "python-embed",
  "scriptExtractDir": "scripts",
  "pthFile": "python311._pth",
  "pythonInteriorZip": "python311.zip",
  "installerRequirements": "",
  "requirementsFile": "requirements.txt",
  "scriptDir": "scripts",
  "setupScript": "",
  "mainScript": "main.py",
  "filesToCopyToRoot": ["requirements.txt", "readme.md", "license.md"],
  "runAfterInstall": false
}
```

**Community and Support**

* **Project Repository**: [https://github.com/IRSS-UBC/Exepy](https://github.com/IRSS-UBC/Exepy)