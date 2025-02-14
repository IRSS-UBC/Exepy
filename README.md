**Exepy "Ex-eh-pie"** 

A straightforward public domain tool for packaging your Python projects into standalone executable files. Inspired by PyInstaller, Exepy offers an alternative built with Golang.

**Why Exepy?**

* **Effortless Distribution:** Share your Python applications with anyone, even users without Python installed. Exepy handles all dependencies for you.
* **Offline Compatibility:**  Your packaged executables will work without needing an active internet connection, as all required libraries are bundled within.
* **Unrestricted Use:**  Exepy's public domain license (the Unlicense) allows you to copy, modify, and distribute your packaged projects for both personal and commercial purposes without limitations.

**Getting Started**

1. **Download Exepy:** Obtain the [latest release](https://github.com/IRSS-UBC/Exepy/releases).
2. **Structure Your Project:**
    * **Script Folder:** Place all your Python code into a designated folder named "payload".
    * **Requirements:** List your project's dependencies in a `requirements.txt` file. If you haven't already, use `pip freeze > requirements.txt` to generate this file.
    * **Configuration (Optional):** Fine-tune Exepy's behavior with a `settings.json` file (explained below). 

3. **Build Your Executable:** Run the Exepy binary, and it will seamlessly bundle your Python environment and scripts into a single, ready-to-distribute executable file.

**Customization (settings.json)**

Exepy offers flexibility through its `settings.json` file. Here's a breakdown of the options:

*  **`pythonDownloadURL`:**  Specify the URL to download the embeddable Python distribution.
*  **`pipDownloadURL`:** URL for downloading the pip installer.
*  **`pythonDownloadFile`:** The filename of the downloaded Python distribution.
*  **`pythonExtractDir`:** The name of the folder where the Python distribution will be extracted.
*  **`pthFile`, `pythonInteriorZip`:** Settings related to internal handling of Python environments
*  **`requirementsFile`:**  The name of your requirements file (defaults to `requirements.txt`).
*  **`payloadDir`:**  The name of the folder containing your Python scripts.
*  **`setupScript`:**  The name of an optional setup script to execute before packaging.
*  **`payloadScript`:**  The name of your primary Python script to be launched by the executable.

**Community and Support**

* **Project Repository** : [https://github.com/IRSS-UBC/Exepy](https://github.com/IRSS-UBC/Exepy)
