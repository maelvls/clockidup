# Standup for Clockify

Install:

```sh
(cd && GO111MODULE=on go get github.com/maelvls/standup@latest)
```

Usage:

```sh
# Login with Clockify:
% standup login

# Print your standup entry for yesterday:
% standup yesterday
- [0.50] prod/cert-manager: cert-manager standup
- [0.74] prod/cert-manager: reviewing PR 3574
- [0.94] prod/cert-manager: cert-manager v1.2-alpha.1 release with irbe and maartje
- [0.39] prod/cert-manager: cert-manager-dev biweekly meeting
```
