# app-sign

## How it works

### Custom Keychain

app-sign creates a separate keychain every time it is started on the Mac using the `security create-keychain` command.
this is cool for CI use so your profiles don't hang around or need to be installed.

### Codesigning

For codesigning it will walk the filetree and execute the `codesign` command for every .app, .appex, .xctest and
.framework directory it can find. Codesign invokations will use the custom keychain that was config'd.

## Troubleshooting

### Make sure certificate and profile are not installed in the default keychain on the mac

The Mac machine must not have the profile or certificate already installed. As the standard keychain is in the keychain
search list, it might happen that the standard keychain is used in that case. That can result in a prompt showing up for
the password or unlocking the standard keychain.

### Other resources

## Great article about codesigning:

https://www.objc.io/issues/17-security/inside-code-signing/
