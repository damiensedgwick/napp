# Napp

Napp is a command line tool that helps you build and test web app ideas *blazingly-fast* with a 
streamlined Go, HTMX, and SQLite stack.

## Table of Contents
- [About](#about)
- [Prerequisites](#prerequisites)
- [Installation](#installation)
- [Usage](#usage)
- [Contributing](#contributing)
- [License](#license)

## About
**What is a Nano App?**
A Nano App is the name I dedicded to give a particular type of application that I am interested in
researching and developing. The constraints are quite simple, it must have as few files as possible
to function correctly, it can be containerised with as small image as possible and it must use as
little JavaScript as possible.

If you want to take a look at how a Nano App is structured by napp, please vist [Napp Template](https://github.com/damiensedgwick/napp-template)

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
`go install github.com/damiensedgwick/napp@latest`

## Usage

`napp <project-name>`

`cd <project-name>`

`go mod init <your-chosen-path>`

`go mod tidy`

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
