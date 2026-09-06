# Release Notes

## Version

v1.0.0

## Release Date

2026-09-06

---

## Features

-  **web:**Add activity audit trail with query api
-  **web:**Honor log level and format with request ids
-  **web:**Add about page with live git tag and header badge
-  **web:**Resolve version from live git tag when unstamped
-  **web:**Stamp version via make and report it at runtime
-  **cli:**Stamp version on every make build
-  **cli:**Update version service to include author information
-  **cli:**Enhance command structure and output formatting
-  **cli:**Add author information to build metadata
-  **ui:**Enhance admin templates with improved layouts and interactivity
-  **storage:**Add variable counting and pagination functionality
-  **assets:**Add SVG logo for branding
-  **api:**Implement variable import functionality and pagination for environment variables
-  **ui:**Update templates with improved layout and app name display
-  **web:**Include app data in environment variables page
-  **auth:**Write initial API key to file and add logout endpoint
-  **web:**Add logout button to header in layout template
-  **web:**Add base HTML layout templates for API and login pages
-  **cli:**Add Docker support for building and running the CLI
-  **cli:**Implement CLI with login, config, and CRUD commands
-  **web:**Add HTML templates for admin dashboard
- Add core web service with API, storage, and admin UI

## Bug Fixes

-  **web:**Single-line header brand with about link
-  **web:**Harden id parsing, deletes, and request context
-  **storage:**Return not found on missing delete
-  **cli:**Validate api base and preserve server errors
-  **cli:**Fail safe on bad config and invalid ids
-  **admin:**Validate env name and show error on create
- Remove stray BOM character from README.md

## Documentation

-  **site:**Add official website in docs folder
- Document operations logs audit and fluent-bit
- Switch license to apache 2.0 like surisc
- Note live git tag version
- Note version-stamped builds
- Rewrite readme in surisc format
- Add AGENTS.md operating manual
- Add community standards guidelines and other documentation files
- Create LICENSE
- Create CODE_OF_CONDUCT.md
- Fix encoding issue in README header
- Fix encoding in README.md header
- Update README with badges and improved formatting
- Update README with consistent bash code block formatting
- Add example environment configuration file
-  **cli:**Add README with build, usage, and command instructions
-  **web:**Add comprehensive README for web server and admin dashboard
- Add initial README with project overview and usage instructions

## Refactoring

- Simplify template rendering and update login page title
-  **api:**Simplify template loading and unify layout rendering

## Style

- Remove unnecessary blank line in import block
-  **ui:**Improve visual design and usability across admin templates
-  **web:**Improve template readability and add error handling
-  **ui:**Update admin dashboard templates with modern design

## Build

-  **cli:**Add Makefile for build automation and versioning
- Add Go module files for CLI tool
- Add Docker and docker-compose configuration for web service
-  **web:**Add air configuration for live reload during development

## CI/CD

-  **web:**Add release workflow for docker image
-  **cli:**Add release workflow for multi-platform binaries
- Add release workflow for cross-platform builds

## Tests

-  **web:**Cover activity log storage and api
-  **web:**Cover public version endpoint
-  **web:**Cover delete not found and invalid ids
-  **cli:**Cover client validation and strict id parsing
- Add unit tests for CLI commands and internal components
- Add unit and integration tests for auth, api, and storage

## Maintenance

- Update .gitignore to include envious_api_key.txt
- Add echo statements to Makefile targets for better visibility
- Add silent execution and verbose output to makefile targets
- Update air configuration for new server binary and directories
-  **web:**Add makefile for common development tasks
-  **web:**Initialize Go module with web framework dependencies
-  **web:**Add .gitignore file for web project
-  **cli:**Add .gitignore file for binary and build artifacts

## Contributors

- Mark Wayne Menorca (49 commits)
- Mark Wayne Buncaras Menorca (25 commits)

