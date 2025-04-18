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
- [Contributors](#contributors)
- [License](#license)

## About

**What is a Nano App?**
The rules are simple, a nano app must have as few files as possible, as few directories as possible,
little to no JavaScript (minus HTMX), has a small Docker image (size does matter) and must be easy
to develop and deploy.

If you want to take a look at how a Nano App is structured by napp, please vist [Napp Generated](https://github.com/damiensedgwick/napp-generated)

Below are some potential use cases for a Nano App.

**UI/UX Experimentation** - Use HTMX's dynamic capabilities to experiment with different user events.

**Proof of Concepts** - Create a lightweight app to showcase the feasibility of a technical concept.

**Admin Interfaces** - Develop basic admin panels to manage users, or config settings for internal systems.

**Go Practice** - Use Napp as a framework to build small projects and improve your Go and web skills.

**HTMX Exploration** - Explore the power of HTMX for building reactive interfaces with minimal JavaScript.

## Prerequisites

- Go 1.23 or higher (It may work on older versions of Go, I developed it using this version).
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

### Docker

Docker has been setup is so that the binary is prebuilt using Go and then it is simply
copied into the Docker image, resulting in a smaller footprint the final Docker image.

**NB** The Dockerfile is currently setup to build for Linux, if your OS is not Linux
you will need to change it accordingly.

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

3. Deploy your application `fly deploy`

This command should now deploy your application to Fly.io and create the
necessary resources (such as our volume) in the process.

4. List Machines to check attached volumes: `fly machine list`

```bash
# expected output
my-app-name
ID              NAME        STATE   REGION   IMAGE                  IP ADDRESS                      VOLUME                  CREATED                 LAST UPDATED            APP PLATFORM    PROCESS GROUP   SIZE
328773d3c47d85  my-app-name stopped yul     flyio/myimageex:latest  fdaa:2:45b:a7b:19c:bbd4:95bb:2  vol_6vjywx86ym8mq3xv    2023-08-20T23:09:24Z    2023-08-20T23:16:15Z    v2              app             shared-cpu-1x:256MB
```

5. List volumes to check attached Machines: `fly volumes list`

```bash
# expected output
ID                      STATE   NAME    SIZE    REGION  ZONE    ENCRYPTED       ATTACHED VM     CREATED AT
vol_zmjnv8m81p5rywgx    created data    1GB     lhr     b6a7    true            5683606c41098e  3 minutes ago
```

6. Add your environment variables to Fly.io

You will need to add the following variables to your app in the Fly.io dashboard.
First navigate to your machine and then to secrets.

Add the following 2 secrets:

`APPNAME_NAME_COOKIE_STORE_SECRET="value"` or `APP_NAME_NAME_COOKIE_STORE_SECRET="value"`

and

`APPNAME_NAME_DB_PATH="value"` or `APP_NAME_DB_PATH="value"`

**NB** Check your generated `.env` file to see how your app name should be
written and add your database path value and your cookie store secret value.

Once these have both been added, we should be ready to deploy for the last time!

6. Run the following command to deploy your app and make use of your .env variables

`fly deploy`

🎉 That should be all you need to do to get your nano app deployed on Fly.io with persisted storage. 🎉

## Contributing

I'd love to have your help making Napp even better! Here's how to get involved:

- Fork the repository. This creates your own copy where you can work on changes.
- Create a separate branch for your changes. This helps keep your work organised.
- Open a pull request with a clear description of your contributions.

## Contributors

A huge shoutout to the following for contributing towards Napp and making it all that
little bit better!

- [Dmytro Borshcanenko](https://github.com/parMaster)

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
