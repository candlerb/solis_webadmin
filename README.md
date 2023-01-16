# Solis web admin

An experimental simple web interface to control a few key parameters in the
Solis 5G hybrid inverter, such as configuring timed charge.

The backend makes a TCP modbus connection, which can connect to the
inverter's RS485 interface using
[solis_exporter](https://github.com/candlerb/solis_exporter) or a generic
modbus TCP gateway.

It should be easy to hack for other use cases.

## Modbus over HTTP??

I don't know if Modbus over HTTP is A Thing™, but the backend is basically a
Modbus-HTTP to Modbus-TCP proxy.  It accepts a POST of a raw modbus message
(without the 7-byte header), forwards it as regular Modbus TCP, and returns
the response in the HTTP body.

The frontend is in Javascript and builds ArrayBuffers containing raw modbus
messages.  This makes it easy to hack on.

However, this is obviously very dangerous.  Make sure this web interface is
blocked from the outside world!

I run it behind an Apache reverse proxy, with mod_auth_openidc talking to a
[Vault OpenID Connect provider](https://brian-candler.medium.com/using-vault-as-an-openid-connect-identity-provider-ee0aaef2bba2),
and HTTPS with a LetsEncrypt certificate.
