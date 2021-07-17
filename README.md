# clockidup, the CLI for generating your standup entry using Clockify

The `clockidup` CLI helps you generating your standup entry using the time entries from [Clockify](https://clockify.me). Imagine that you recorded your activity and your Clockify day looks like this:

![example-clockify](https://user-images.githubusercontent.com/2195781/106798923-ef23ec80-665e-11eb-8810-c023b00a2c14.png)

The `clockidup` CLI allows you to generate the following standup message:

```md
% clockidup today
Wednesday:

- [.8] prod/cert-manager: cert-manager standup
- [1.2] prod/cert-manager: triaging #2037: HTTPS support for the solver’s listener
- [4.6] prod/cert-manager: preparing for 1.2: get #3505 merged
- [4.1] prod/cert-manager: work on jsp-gcm: add manual test for cas-issuer
```

## Features

- User-friendly `login` command for setting up and remembering the Clockify token;
- De-duplication of time entries that have the same description;
- Supports the "billable" property,
- Support switching between Clockify workspaces,
- Supports tasks (see below).

## Installation

```sh
go install github.com/maelvls/clockidup@latest
```

You can also use the pre-built binaries available on the [GitHub
Release](https://github.com/maelvls/clockidup/releases) page (for Linux and
macOS).

[![GitHub tag](https://img.shields.io/github/release/maelvls/clockidup.svg)](https://github.com/maelvls/clockidup/releases)

## Usage

To login with Clockify:

```sh
clockidup login
```

![](https://user-images.githubusercontent.com/2195781/123278842-95d23200-d507-11eb-8d31-7678575e8d37.gif)

If you are using multiple Clockify workspaces, you can also switch between
workspaces using `clockidup select`

```sh
% clockidup select
? Choose a workspace  [Use arrows to move, type to filter]
  jetstack-mael-valais
> personal
```

You can print your standup message for yesterday:

```md
% clockidup yesterday
Thursday:

- [.5] prod/cert-manager: cert-manager standup
- [.7] prod/cert-manager: reviewing PR 3574
- [1.0] prod/cert-manager: cert-manager v1.2-alpha.1 release with irbe and maartje
- [.4] prod/cert-manager: cert-manager-dev biweekly meeting
```

You can also print today's message:

```md
% clockidup today
Monday:

- [2.8] no-project: easy-slack-oauth
```

> Note: I did not set the project for today's entries, which is an invalid standup message; in this case, clockidup will show `no-project`.

You can even use an arbitrary human-readable relative date as specified by [tj/go-naturaldate](https://github.com/tj/go-naturaldate#examples). Some examples:

```sh
clockidup tuesday
clockidup "4 days ago"
clockidup "28 Jan"
```

Note that the following commands are equivalent:

```sh
clockidup tuesday
clockidup "last tuesday"
```

Tasks in Clockify are also supported and can be optionally used. If a time entry has an associated task, the task name will prefix the entry description. Imagine that you have a time entry "fix progress bar" and this time entry is linked to a broader task "download feature", then the output will be:

```md
- [0.50] prod/my-super-product: download feature: add progress bar
  <----project name----> <---task name---> <--entry name-->
```

The date printed on the first line is compatible with the expected standup date format. It either shows the week day e.g. "Monday" if the given day is within the past week and "2021-01-28" if not.

For example, when the day is within the week:

```md
% clockidup today
Wednesday:

- [.8] prod/cert-manager: cert-manager standup
- [1.2] prod/cert-manager: triaging #2037: HTTPS support for the solver’s listener
- [4.6] prod/cert-manager: preparing for 1.2: get #3505 merged
- [4.1] prod/cert-manager: work on jsp-gcm: add manual test for cas-issuer
```

And when the day is more than 7 days ago, you can use the
[ISO 8601, calendar date](https://en.wikipedia.org/wiki/ISO_8601#Calendar_dates) format as input (as of v0.2.0):

```md
% clockidup 2021-01-27
2021-01-27:

- [2.6] prod/cert-manager: work on jsp-gcm
- [.6] admin: coffee with Mattias Gees
- [.0] prod/cert-manager: cert-manager standup
- [.7] prod/cert-manager: reviewing josh comments on PR 3574
- [.1] prod/cert-manager: review of Design: cert-manager certificates.k8s.io Adoption
- [.1] prod/cert-manager: review of Design: cert-manager Identity
- [.1] prod/cert-manager: review of Design cert-manager Policy
- [1.6] prod/cert-manager: investigate #3603 "multiple certificaterequests"
- [.9] prod/cert-manager: cert-manager v1.2-alpha.1 release with irbe and maartje
- [.6] prod/cert-manager: cert-manager internal meeting
- [.4] prod/cert-manager: cert-manager-dev biweekly meeting
```

You can use the `--billable` flag if you only want to see the Clockify entries
that have the `billable: true` property.

<div style="text-align: right"><a href="https://github.com/maelvls/clockidup/edit/main/README.md">🐓 Edit this page</a></div>
