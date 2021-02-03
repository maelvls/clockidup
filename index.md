The `clockidup` CLI helps you generating your standup entry using the time entries from [Clockify](https://clockify.me).

Features:

- User-friendly `login` command for setting up and remembering the Clockify token;
- De-duplication of time entries that have the same description;
- Supports tasks (see below).

## Installation

```sh
(cd && GO111MODULE=on go get github.com/maelvls/clockidup@latest)
```

[![GitHub tag](https://img.shields.io/github/release/maelvls/clockidup.svg)](https://github.com/maelvls/clockidup/releases)

## Usage


To login with Clockify:

```sh
clockidup login
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


For example, when the day is within the week:

```md
% clockidup today
Wednesday:
- [.8] prod/cert-manager: cert-manager standup
- [1.2] prod/cert-manager: triaging #2037: HTTPS support for the solver’s listener
- [4.6] prod/cert-manager: preparing for 1.2: get #3505 merged
- [4.1] prod/cert-manager: work on jsp-gcm: add manual test for cas-issuer
```

And when the day is more than 7 days ago:

```md
% clockidup wednesday
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
```md

