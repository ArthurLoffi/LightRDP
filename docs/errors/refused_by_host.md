# Refused by the host
**Message:** `error: connection refused: refused by host`

## Description
The person at the host refused your connection request.

## Cause
When the two computers are connected, the host sees the safety code and is asked `Accept "<YOUR_ID>" with this code? [y/N]`. Any answer other than `y`, including just pressing Enter, refuses the request. The person at the host may also have refused because the code on their screen was different from yours.

## Possible solution
- Confirm with the person at the host that they expect your connection.
- Try again, compare the safety code with them, and ask them to answer `y` if it is the same.
- If the codes are different, do not use that network or that server. See [Safety code not confirmed](safety_code_not_confirmed.md).

_Documented on: 2026-11-16 (updated 2026-11-18)_
