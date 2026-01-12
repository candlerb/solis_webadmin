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

The web interface by default binds to `127.0.0.1:8502`, so it only accepts
connections from the local host.  If you really want to expose it directly
to network traffic, then you'll need to supply a flag like `-listen :8502`

## solis_boost

A secondary utility which programmes timed charges via TCP modbus, in
response to messages from Home Assistant saying that off-peak rates are
active.
