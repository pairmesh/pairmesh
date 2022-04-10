# HOW TO CONTRIBUTE

## Contribution Guide

PairMesh is a community-driven open source project and we look forward to you being part of the community of contributors. Contributions to the PairMesh project are expected to adhere to our [Code of Conduct](https://github.com/pairmesh/pairmesh/blob/master/CODE_OF_CONDUCT.md).

This document outlines some conventions about development workflow, commit message formatting, contact points and other resources to make it easier to get your contribution accepted. You can also join us in our [Slack](https://pairmesh.slack.com) for help with any issues.

<!-- TOC -->

- [Setting up your development environment](#setting-up-your-development-environment)
- [Your First Contribution](#your-first-contribution)
- [Before you open your PR](#before-you-open-your-pr)
- [Contribution Workflow](#contribution-workflow)
- [Style reference](#style-reference)
- [Get a code review](#get-a-code-review)

<!-- /TOC -->

## Setting up your development environment

PairMesh is written in GO. Before you start contributing code to PairMesh, you need to
set up your GO development environment.

1. Install `Go` version **1.17** or above. Refer to [How to Write Go Code](http://golang.org/doc/code.html) for more information.
2. Define `GOPATH` environment variable and modify `PATH` to access your Go binaries. A common setup is as follows. You could always specify it based on your own flavor.

    ```sh
    export GOPATH=$HOME/go
    export PATH=$PATH:$GOPATH/bin
    ```

3. PairMesh uses [`Go Modules`](https://github.com/golang/go/wiki/Modules)
to manage dependencies.

Now you should be able to use the `make` command to build PairMesh.

## Your First Contribution

All set to contribute? You can start by finding an existing issue with the
[help wanted](https://github.com/pairmesh/pairmesh/issues?q=is%3Aopen+is%3Aissue+label%3A%22help+wanted%22) and [godd first issue](https://github.com/pairmesh/pairmesh/issues?q=is%3Aopen+is%3Aissue+label%3A%22good+first+issue%22) labels in the PairMesh repository. These issues are well suited for new contributors.

## Before you open your PR

Before you move on, please make sure what your issue and/or pull request is, a
simple bug fix or an architecture change.

In order to save reviewers' time, each issue should be filed with template and
should be sanity-checkable in under 5 minutes.

- **Is this a simple bug fix?**

    Bug fixes usually come with tests. With the help of continuous integration
    test, patches can be easy to review. Please update the unit tests so that they
    catch the bug!

- **Is this an architecture improvement?**

    Some examples of "Architecture" improvements:

    - Converting structs to interfaces.
    - Improving test coverage.
    - Decoupling logic or creation of new utilities.
    - Making code more resilient (sleeps, backoffs, reducing flakiness, etc).

    If you are improving the quality of code, then justify/state exactly what you
    are 'cleaning up' in your Pull Request so as to save reviewers' time.

    If you're making code more resilient, test it locally to demonstrate how
    exactly your patch changes things.

    > **Tip:**
    >
    >To improve the efficiency of other contributors and avoid
    duplicated working, it's better to leave a comment in the issue that you are
    working on.

## Contribution Workflow

To contribute to the PairMesh code base, please follow the workflow as defined in this section.

1. Create a topic branch from where you want to base your work. This is usually master.
2. Make commits of logical units and add test case if the change fixes a bug or adds new functionality.
3. Run tests and make sure all the tests are passed.
4. Make sure your commit messages are in the proper format (see below).
5. Push your changes to a topic branch in your fork of the repository.
6. Submit a pull request.

This is a rough outline of what a contributor's workflow looks like. For more details, see [GitHub workflow](./docs/guide/github-workflow.md).

Thanks for your contributions!

## Get a code review

If your pull request (PR) is opened, it will be assigned to reviewers within the relevant Special Interest Group (SIG). Normally each PR requires at least 1 LGTMs (Looks Good To Me) from eligible reviewers. Those reviewers will do a thorough code review, looking at correctness, bugs, opportunities for improvement, documentation and comments,
and style.

To address review comments, you should commit the changes to the same branch of
the PR on your fork.

### Style reference

Keeping a consistent style for code, code comments, commit messages, and pull requests is very important for a project like PairMesh. We highly recommend you refer to the [style guide](https://github.com/uber-go/guide/blob/master/style.md) for details.

