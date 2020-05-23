![License](https://img.shields.io/github/license/milgner/keedrop)
![Build](https://github.com/milgner/keedrop/workflows/Build/badge.svg)
![Uptime Ratio](https://img.shields.io/uptimerobot/ratio/m785003856-8aa9823b6992a1796824c5a6)
[![Maintainability](https://api.codeclimate.com/v1/badges/84247dcbbb258baa2cf0/maintainability)](https://codeclimate.com/github/milgner/keedrop/maintainability)
![Chat](https://img.shields.io/matrix/keedrop:matrix.illunis.net)

# KeeDrop - securely send passwords

Ever wanted to securely send a password to someone who didn't have GnuPG?
Enter **KeeDrop**: each secret can only be retrieved once and only for 24h after
it was stored. Additionally, it is encrypted in a way that ensures that the
server operator will not be able to decrypt the password.
The only way to be more secure is to use GnuPG or other end-to-end encryption
with key verification.

## Security features

- State-of-the-art encryption technology: Curve25519-XSalsa20-Poly1305, courtesy of [TweetNaCl](http://tweetnacl.js.org/).
- The server never sees the unencrypted secret
- Automatically expunge data after 24h
- Alternatively, data is deleted on retrieval
- Doesn't track you with analytics

## What it cannot defend against:

- Malicious software (browser extensions, keylogger etc) on the senders or recipients computer.
- Someone intercepting the link and retrieving the secret. But at least you'll know about it.
