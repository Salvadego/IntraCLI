# IntraCLI

**IntraCLI** is a command-line interface for interacting with **Mantis**, the
issue and project management system. It provides a framework for integrating
with Mantis APIs, currently optimized for **timesheet management**, but
designed to expand into other areas of Mantis operations.

## Table of Contents
  - [Overview](#overview)
  - [Features](#features)
  - [Installation](#installation)
  - [Getting Started (Required Setup)](#getting-started-required-setup)
    - [Step 1 – Search for an Employee (Required)](#step-1-search-for-an-employee-required)
    - [Step 2 – Verify Assigned Projects](#step-2-verify-assigned-projects)
    - [Step 3 – Create Appointments](#step-3-create-appointments)
  - [CLI Usage](#cli-usage)
  - [Shell Completion](#shell-completion)
    - [Enabling Completion](#enabling-completion)
      - [Bash](#bash)
      - [Zsh](#zsh)
      - [Fish](#fish)
    - [Completion Features](#completion-features)
    - [Profiles](#profiles)
    - [Timesheets (Core Feature)](#timesheets-core-feature)
    - [Filters](#filters)
    - [Projects](#projects)
    - [Roles](#roles)
  - [Examples](#examples)
  - [Caveats](#caveats)
  - [Extending IntraCLI](#extending-intracli)


---

## Overview

IntraCLI connects to Mantis via its API and allows operations such as:

* Viewing and editing timesheets.
* Filtering and summarizing work entries.
* Managing project aliases.
* Searching employees and managing profiles.
* Modifying roles for API actions.

While timesheets are fully implemented and optimized, the CLI is designed as a **platform for broader Mantis integration**, allowing future expansion into issues, projects, and reporting.

---

## Features

* **Timesheets:** List, filter, edit, batch update, and undo operations.
* **Filters:** Save reusable timesheet filters and daily summaries.
* **Profiles:** Multi-user support, switch profiles, associate employees.
* **Projects:** List projects and assign aliases for faster timesheet creation.
* **Roles:** View and modify Mantis roles.
* **Extensible CLI:** Framework ready to integrate additional Mantis endpoints.

---

## Installation

```bash
git clone https://github.com/Salvadego/IntraCLI.git
cd IntraCLI
go build -o intracli main.go
mv intracli /usr/local/bin/  # optional
```

Verify installation:

```bash
intracli --help
```

---

## Getting Started (Required Setup)

IntraCLI performs an automatic bootstrap on the first real command you run.

During this bootstrap, the CLI will:

* Create the configuration file if it does not exist
* Ask for Mantis credentials and base URL
* Ensure a default profile exists (empty or partially populated)

After bootstrap, the configuration exists — but **you are still not ready to
create appointments yet**.

Important: To create appointments, the user must first search for an employee
and have at least one asigned aliased project.

This step is mandatory and must be done explicitly.

---

### Step 1 – Search for an Employee (Required)

Before you can:

* Create appointments
* List assigned projects
* Add project aliases
* Edit or delete timesheets

You must associate your profile with a real employee in Mantis.

Run:

```bash
intracli search-employee --name "John Doe" --create-profile myprofile
```

This command:

* Searches Mantis for matching employees
* Retrieves the employee ID and employee code
* Populates the profile with the required data
* Enables project discovery and timesheet operations

Without this step:

* list-projects will fail
* appoint will fail
* edit-timesheet will fail
* Any operation requiring projects or employee data will fail

---

### Step 2 – Verify Assigned Projects

Once the employee is associated with the profile, verify that the user has projects assigned in Mantis.

Run:

```bash
intracli list-projects
```

If no projects are returned, the user cannot create appointments.
Projects must be assigned in Mantis first.

```bash
intracli list-projects -n project-number -a alias_name
```

---

### Step 3 – Create Appointments

Only after:

* Bootstrap is completed
* An employee is associated with the profile
* At least one project is assigned

You can create appointments.

Example:

```bash
intracli appoint --project-alias PROJX --hours 8 --description "Worked on feature X"
```

You can see other things using the help command for any command:

```bash
intracli help appoint
```

## CLI Usage

## Shell Completion

IntraCLI supports **shell completion** for commands, flags, and profiles via
**Cobra**. This helps speed up CLI usage and reduces errors in typing profile
names, filter names, and project aliases.

### Enabling Completion

#### Bash

This requires [BASH COMPLETION](https://github.com/scop/bash-completion)

##### Install based on your package manager
```bash
sudo apt install bash-completion
```

##### Use bash-completion, if available, and avoid double-sourcing
```bash
[[ $PS1 &&
  ! ${BASH_COMPLETION_VERSINFO:-} &&
  -f /usr/share/bash-completion/bash_completion ]] &&
    . /usr/share/bash-completion/bash_completion
```

```bash
echo "source <(intracli completion bash)" >> ~/.bashrc
```

#### Zsh

```bash
echo "source <(intracli completion zsh)" >> ~/.zshrc
```

#### Fish

```bash
intracli completion fish > ~/.config/fish/completions/intracli.fish
```

### Completion Features

* Command and subcommand completion (`list-timesheets`, `edit-timesheet`, `filter-timesheets`, etc.)
* Flag completion (`--profile`, `--filter`, `--project-alias`, `--id`, etc.)
* Dynamic completion for profiles, saved filters, project aliases, and Mantis roles.

Example:

```bash
intracli edit-timesheet --filter <TAB>
# Autocompletes saved filter names
```

---

### Profiles

* **Search employee and create profile:**

```bash
intracli search-employee --name "John Doe" --create-profile myprofile
```

* **Switch profile:**

```bash
intracli --profile myprofile list-timesheets
```

---

### Timesheets (Core Feature)

* **List timesheets:**

```bash
intracli list-timesheets --year 2025 --month 10
intracli list-timesheets --filter myfilter
```

* **Edit timesheets (batch or single):**

```bash
intracli edit-timesheet --id 123 --hours 4 --description "Updated task"
intracli edit-timesheet --filter myfilter --hours 6
```

* **Undo last deletion or edit:**

```bash
intracli undo-timesheet
```

* **Summary generation:**

```bash
intracli date-summary
```

---

### Filters

* **Create/save filter:**

```bash
intracli filter-timesheets --save myfilter --project PROJX --from 2025-01-01 --to 2025-01-31 --has-ticket-only
```

* **List filters:**

```bash
intracli filter-timesheets --list
```

* **Delete filter:**

```bash
intracli filter-timesheets --delete myfilter
```

* **Daily filters:** (min hours, project, user, status)

```bash
intracli filter-days --save minhours --from 2025-01-01 --to 2025-01-31 --min-hours 8
intracli filter-days --list
```

---

### Projects

* **List projects assigned to user:**

```bash
intracli list-projects
```

* **Add project alias:**

```bash
intracli list-projects --alias PROJX --project-number 101
```

---

### Roles

* **List roles:**

```bash
intracli roles
```

* **Modify profile role (interactive):**

```bash
intracli roles --modify
```

---

## Examples

* **Batch update hours for a project:**

```bash
intracli edit-timesheet --filter PROJXFilter --hours 7.5
```

* **Filter timesheets with regex on description:**

```bash
intracli filter-timesheets --save regexfilter --description "^Meeting.*"
```

* **Daily summary:**

```bash
intracli daily-summary --filter minHours
```

---

## Caveats

* `edit-timesheet` deletes and recreates entries. Undo uses local cache; failure may cause permanent loss.
* Quantity filters require format: `>=2.5`, `<=8`, `=4`.
* Dates must be in `YYYY-MM-DD` format.
* Project alias matching is exact.
* Role modification is interactive and cannot be scripted.

---

## Extending IntraCLI

The CLI is designed to support additional Mantis operations:

* **Issues:** create, update, list, and filter.
* **Projects:** manage project info, team members, deadlines.
* **Reports:** generate custom reports from timesheets and issues.

The `mantisClient` object abstracts Mantis API calls. New commands can be
added via `cobra` and use existing utilities for filtering, caching, and
summaries.

---
