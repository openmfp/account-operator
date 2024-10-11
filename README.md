> [!WARNING]
> This Repository is under construction and not yet ready for public consumption. Please check back later for updates.


# openMFP - account-operator
![Build Status](https://github.com/openmfp/account-operator/actions/workflows/pipeline.yml/badge.svg)

## Description

The openMFP account-operator manages the core Account resource which is a grouping entity in openMFP. It manages a related Namespace and will instantiate additional configured resources in its owned Namespace.

## Features
- Account Namespace management
- Instantiation of Account Resource in Namespace
- Support for Spreading Reconciles to improve performance on operator restart
- Validating webhook to ensure that immutable information is not changed
- Cleanup on Account deletion including namespace cleanup

## Getting started

TBD

## Releasing

The release is performed automatically through a GitHub Actions Workflow.

All the released versions will be available through access to GitHub (as any other Golang Module).

## Requirements

The account-operator requires a installation of go. Checkout the [go.mod](go.mod) for the required go version and dependencies.

## Contributing

Please refer to the [CONTRIBUTING.md](CONTRIBUTING.md) file in this repository for instructions on how to contribute to openMFP.

## Code of Conduct

Please refer to the [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md) file in this repository informations on the expected Code of Conduct for contributing to openMFP.

## Licensing

Copyright 2024 SAP SE or an SAP affiliate company and openMFP contributors. Please see our [LICENSE](LICENSE) for copyright and license information. Detailed information including third-party components and their licensing/copyright information is available [via the REUSE tool](https://api.reuse.software/info/github.com/openmfp/account-operator).
