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
- [Deployment](#deployment)
- [Contributing](#contributing)
- [License](#license)

## About
**What is a Nano App?**
The rules are simple, a nano app must have as few files as possible, as few directories as possible,
little to no JavaScript (minus HTMX), has a small Docker image (size does matter) and must be easy
to develop and deploy.

If you want to take a look at how a Nano App is structured by napp, please vist [Napp Generated](https://github.com/damiensedgwick/napp-generated)

Below are some potential use cases for a Nano App.

**UI/UX Experimentation** - Use HTMX's dynamic capabilities to experiment with different user interface
flows and interactions without heavy JavaScript reliance.

**Proof of Concepts** - Create lightweightapps to showcase the feasibility of a technical concept for
clients or team members.

**Admin Interfaces** - Develop basic admin panels to manage users, content, or configuration settings for
internal systems.

**Go Practice** - Use Napp as a framework to build small projects and improve your Go coding skills and
web development techniques.

**HTMX Exploration** - Explore the power of HTMX for building reactive interfaces with minimal JavaScript.

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

## Deployment

At the moment I recommend using Fly.io for deploying Nano Apps. They provide a great
command line tool to make managing the process easy and you can have 3 x small projects
on there each with a 1gb persisted SQLite database volume for prototyping.

### Getting your Nano App Deploy on Fly.io

You will need to make sure you have the Fly.io command line tools installed for the
following steps. If you do not have them installed, you can find out how to install them
here: [flyctl command line tool](https://fly.io/docs/hands-on/install-flyctl/)

Once you have installed the above command line tools and signed in to Fly.io, you are
ready to proceed with the next steps, each step is supposed to be run from your project
root.

1. Get your app ready for deploying: `fly launch --no-deploy`

The command line will prompt you to check if you would like to change the default
configuration. It is important to say yes so that you can select your region and
lower the apps memory to 256mb vs the default settings.

2. Copy the following into your `fly.toml`

```toml
[[mounts]]
  source = "godo_database"
  destination = "/data"
  initial_size = "1gb"
```

3. Create your volume `fly volume create your-app-name -r <region> -n <count> -s <size>`

An example would be `fly volume create your-app-name -r lhr -n 1 -s 1` for 1 volume in
the London region and a size of 1gb.

5. Check your volume is correctect: `fly volumes list`

```bash
# expected output
ID                      STATE   NAME            SIZE    REGION  ZONE    ENCRYPTED       ATTACHED VM     CREATED AT
vol_some-id-number    created   app_data     1GB        lhr     bXXc    true            4573d4857eh34   20 hours ago
```

5. Deploy your app `fly deploy --ha=false`

Providing that I have not forgot anything, that should be all you need to do to get
your nano app deployed on Fly.io with some persisted storage.

6. You will need to add 2 environment variables to your Fly application. As they are
dynamic and specific to your application I am not able to tell you what they are.

They should however look something like this

`SOME_NAME_COOKIE_STORE_SECRET`

and

`SOME_NAME_DB_PATH`

These will both need to be added to your application for it to run properly once deployed.

## Contributing
I'd love to have your help making Napp even better! Here's how to get involved:

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
