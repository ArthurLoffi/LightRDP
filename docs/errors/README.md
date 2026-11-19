# Error Docs
Every known error in the application, with help on how to solve each one.

## Getting help
If you run into an error that is not documented here, or you cannot solve it, feel free to open an issue or send an email to:
arthuroscarsoares@gmail.com

## Known errors
- [Invalid arguments](invalid_arguments.md)
- [Cannot reach the signaling server](server_unreachable.md)
- [ID already registered](id_already_registered.md)
- [Host not found](peer_not_found.md)
- [Host busy](host_busy.md)
- [Refused by the host](refused_by_host.md)
- [Host did not answer in time](confirmation_timeout.md)
- [Signaling server disconnected](server_disconnected.md)
- [Peer connection failed](peer_connection_failed.md)
- [Insecure server address](insecure_server_url.md)
- [Server certificate not trusted](untrusted_certificate.md)
- [Certificate does not match the server address](certificate_name_mismatch.md)
- [Invalid certificate file](invalid_ca_file.md)
- [Certificate or key missing](tls_flags_incomplete.md)
- [Encryption required on the network](tls_required.md)
- [Server cannot load the certificate](server_tls_files.md)
- [Safety code not confirmed](safety_code_not_confirmed.md)

## For agents
Document each error in its own `.md` file in this folder, and add a link to it in the list above. Each error must have:
- Title
- Description
- Cause
- Possible solution
- Date it was documented
