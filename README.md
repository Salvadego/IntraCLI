# IntraCLI

**IntraCLI** is a command-line interface for interacting with **Mantis**, the
issue and project management system. It provides a framework for integrating
with Mantis APIs, currently optimized for **timesheet management**, but
designed to expand into other areas of Mantis operations.

## Table of Contents
<!--toc:start-->
- [IntraCLI](#intracli)
  - [Table of Contents](#table-of-contents)
  - [Overview](#overview)
  - [Features](#features)
  - [Installation](#installation)
  - [Configuration](#configuration)
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
<!--toc:end-->

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

## Configuration

Configuration is stored in:

```
~/.config/intracli/config.yaml
```

Minimal config:

```yaml
defaultProfile: "myprofile"
profiles:
  myprofile:
    employeeName: "John Doe"
    userID: 123
    email: "john.doe@example.com"
    employeeCode: 456
    dailyJourney: 8.0
    projectAliases: {}
savedFilters: {}
savedDayFilters: {}
roleID: 0
baseURL: "https://mantis.example.com"
```

Profiles contain employee metadata, daily journey, and project aliases for quick timesheet entry.

---

## CLI Usage

## Shell Completion

IntraCLI supports **shell completion** for commands, flags, and profiles via
**Cobra**. This helps speed up CLI usage and reduces errors in typing profile
names, filter names, and project aliases.

### Enabling Completion

#### Bash

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
