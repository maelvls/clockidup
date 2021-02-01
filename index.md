The `clockidup` CLI helps you generating your standup entry using the time entries from [Clockify](https://clockify.me).

Features:

- User-friendly `login` command for setting up and remembering the Clockify token;
- De-duplication of time entries that have the same description.

## Installation

```sh
(cd && GO111MODULE=on go get github.com/maelvls/clockidup@latest)
```

## Usage


To login with Clockify:

```sh
clockidup login
```

You can print your standup message for yesterday:

```
% clockidup yesterday
Thursday, 28 Jan 2021:
- [.5] prod/cert-manager: cert-manager standup
- [.7] prod/cert-manager: reviewing PR 3574
- [1.0] prod/cert-manager: cert-manager v1.2-alpha.1 release with irbe and maartje
- [.4] prod/cert-manager: cert-manager-dev biweekly meeting
```


You can also print today's message:

```
% clockidup today
Monday, 1 Feb 2021:
- [2.8] no-project: easy-slack-oauth
```

> Note: I did not set the project for today's entries, which is an invalid standup message.

You can even use an arbitrary human-readable relative date as specified by [tj/go-naturaldate](https://github.com/tj/go-naturaldate#examples). Some examples:

```
clockidup "last tuesday"
clockidup "4 days ago"
clockidup "28 Jan"
```



