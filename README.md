> [!WARNING]
> This Repository is under development and not ready for productive use. It is in an alpha stage. That means APIs and concepts may change on short notice including breaking changes or complete removal of apis.

# OpenMFP - account-operator
![Build Status](https://github.com/openmfp/account-operator/actions/workflows/pipeline.yml/badge.svg)

## Description

The OpenMFP account-operator manages the core Account resource which is a grouping entity in OpenMFP. It manages a related Namespace and will instantiate additional configured resources in its owned Namespace.

## Features
- Account Namespace management
- Instantiation of Account Resource in Namespace
- Support for Spreading Reconciles to improve performance on operator restart****
- Validating webhook to ensure that immutable information is not changed
- Cleanup on Account deletion including namespace cleanup

## Getting started

TBD

## Releasing

The release is performed automatically through a GitHub Actions Workflow.

All the released versions will be available through access to GitHub (as any other Golang Module).

## Requirements

The account-operator requires a installation of go. Checkout the [go.mod](go.mod) for the required go version and dependencies.

## Security / Disclosure
If you find any bug that may be a security problem, please follow our instructions at [in our security policy](https://github.com/openmfp/extension-manager-operator/security/policy) on how to report it. Please do not create GitHub issues for security-related doubts or problems.

## Contributing

Please refer to the [CONTRIBUTING.md](CONTRIBUTING.md) file in this repository for instructions on how to contribute to OpenMFP.

## Code of Conduct

Please refer to the [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md) file in this repository informations on the expected Code of Conduct for contributing to OpenMFP.

## Licensing

Copyright 2024 SAP SE or an SAP affiliate company and OpenMFP contributors. Please see our [LICENSE](LICENSE) for copyright and license information. Detailed information including third-party components and their licensing/copyright information is available [via the REUSE tool](https://api.reuse.software/info/github.com/OpenMFP/account-operator).

