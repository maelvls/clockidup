The `clockidup` CLI helps you generating your standup entry using the time entries from [Clockify](https://clockify.me).

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
- [0.50] prod/cert-manager: cert-manager standup
- [0.74] prod/cert-manager: reviewing PR 3574
- [0.94] prod/cert-manager: cert-manager v1.2-alpha.1 release with irbe and maartje
- [0.39] prod/cert-manager: cert-manager-dev biweekly meeting
```


You can also print today's message:

```
% clockidup today
- [2.85] : easy-slack-oauth
```

(Note that I did not set the project for today's entries, which is an invalid standup message)
