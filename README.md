# Napp

Napp is an abbreviation of 'Nano App' which is a term I am using to describe a web application
that has a very small footprint. Specifically, Napp bootstraps a new application for you by
creating a project, necessary files and connects them all up so you can dive straight into
building your application.

## Table of Contents
- [About](#about)
- [Prerequisites](#prerequisites)
- [Installation](#installation)
- [Usage](#usage)
- [Contributing](#contributing)
- [License](#license)

## About
**What is a Nano App?**
The rules are simple, a nano app must have as few files as possible, as few directories as possible,
little to no JavaScript (minus HTMX), has a small Docker image (size does matter) and must be easy
to develop and deploy.

If you want to take a look at how a Nano App is structured by napp, please vist [Napp Generated](https://github.com/damiensedgwick/napp-generated)

Below are some potential use cases for a Nano App.

### Rapid Prototyping:

**Idea Validation** - Quickly build a minimal working prototype to test the core
functionality of an app idea. This is great for validating assumptions before investing in a full-scale
project. 

**UI/UX Experimentation** - Use HTMX's dynamic capabilities to experiment with different user interface
flows and interactions without heavy JavaScript reliance.

**Proof of Concepts** - Create lightweightapps to showcase the feasibility of a technical concept for
clients or team members.

### Small-Scale Internal Tools:

**Custom Dashboards** - Build dashboards to monitor internal metrics, system statuses, or visualize data.
SQLite's simplicity makes it perfect for storing and querying relevant information.

**Admin Interfaces** - Develop basic admin panels to manage users, content, or configuration settings for
internal systems.
    
**Workflow Automation** - Create simple tools to automate repetitive tasks, such as data processing or
report generation.

### Personal Projects:

**Habit Trackers** - Build a lightweight web app to track personal habits, goals, or progress (e.g.,
logging workouts, reading, etc.).

**Recipe Books** - Create a personal recipe manager where you can store, search, and categorize your
favorite recipes.

**Simple Note-taking Apps** - Develop a basic notes app for jotting down ideas, to-dos, or reminders.

### Learning and Experimentation:

**Go Practice** - Use Napp as a framework to build small projects and improve your Go coding skills and
web development techniques.

**HTMX Exploration** - Explore the power of HTMX for building reactive interfaces with minimal JavaScript.

**Database Fundamentals** - Learn and apply basic database concepts with SQLite for data storage and retrieval.

## Prerequisites
- Go 1.22 or higher (It may work on older versions of Go, I developed it using this version).
- Make sure your Go bin is added to your path so that installed packages can be used globally.

## Installation

### Using Go
`go install github.com/damiensedgwick/napp@latest`

### Using a release binary
You can download the compiled binary for your machine from [Releases](https://github.com/damiensedgwick/napp/releases).
Once you have done this, make sure the executable has the correct permissions and make sure it is in your path.

## Usage

### Generate a new Napp

`napp init <project-name>`

`cd <project-name>`

`go mod init <your-chosen-path>`

`go mod tidy`

### Other commands

Display the Napp help menu to get a list of currently available commands.

`napp --help`

Get the current version with the following command:

`napp --version`

## Running the application

### Go

Go has everything you need to build and run the application locally and it is
usually the default choice when wanting to develop and iterate quickly.

`go run cmd/main.go`

### Make

I personally like using Make, I think it is simple and does the job well. The following
command will clean, format, build and run the application.

`make all`

### Docker

Docker has been setup is so that the binary is prebuilt using Go and then it is simply
copied into the Docker image, resulting in a smaller footprint the final Docker image.

`docker build -t app-name .`

`docker run -d -p 8080:8080 app-name`

## Contributing
I'd love to have your help making [project name] even better! Here's how to get involved:

- Fork the repository. This creates your own copy where you can work on changes.
- Create a separate branch for your changes. This helps keep your work organised.
- Open a pull request with a clear description of your contributions.

## License
The MIT License (MIT)

Copyright (c) <year> Adam Veldhousen

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
