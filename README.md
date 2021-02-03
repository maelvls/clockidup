# clockidup, the CLI for generating your standup entry using Clockify

Install:

```sh
(cd && GO111MODULE=on go get github.com/maelvls/clockidup@latest)
```

Usage:

```sh
# Login with Clockify:
% clockidup login
```

```
# Print your standup entry for yesterday:
% clockidup yesterday
Wednesday:
- [0.50] prod/cert-manager: cert-manager standup
- [0.74] prod/cert-manager: reviewing PR 3574
- [0.94] prod/cert-manager: cert-manager v1.2-alpha.1 release with irbe and maartje
- [0.39] prod/cert-manager: cert-manager-dev biweekly meeting
```
