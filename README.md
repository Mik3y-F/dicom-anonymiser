# DICOM Anonymiser

## Table of Contents

1. [Introduction](#introduction)
2. [Installation](#installation)
3. [Usage](#usage)
4. [License](#license)
5. [Contact](#contact)

## Introduction

`dicom-anonymiser` is a Go-based REST API designed to anonymise DICOM (Digital Imaging and Communications in Medicine) files. DICOM is a standard used for transmitting, storing, retrieving, and sharing medical imaging information.

This tool is perfect for healthcare providers, researchers, and software developers who need to protect patient information by removing identifiable data from DICOM files in compliance with privacy laws and regulations.

**Please note**: This project is in active development. Use at your own risk.

## Installation

Before you begin, ensure you have met the following requirements:

* You have installed Go 1.18 or later.

* You have a basic understanding of how to use the command line interface.

To install `dicom-anonymiser`, follow these steps:

1. Clone the repository:

    ```sh
    git clone https://github.com/Mik3y-F/dicom-anonymiser.git
    ```

2. Navigate to the project directory:

    ```sh
    cd dicom-anonymiser
    ```

3. Run the project:

    ```sh
    go run ./cmd/dicomd/main.go
    ```

## Usage

To start the server, run:

```sh
./dicom-anonymiser
```

The REST API documentation is available at `http://localhost:8000/swagger-ui/` (default port is `8000`).
(Not available at the moment but will be added)

## License

dicom-anonymiser is licensed under the terms of MIT license. See the LICENSE file for details.

## Contact

If you have any questions, issues, or suggestions, feel free to open an issue on the project's GitHub page or contact the maintainer directly at [mike18farad@gmail.com](mike18faradgmail.com).

Happy anonymising!
