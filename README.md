# chdbg

Based on:
- [chdbg (fork of)](https://github.com/orijtech/chdbg)
- [`iavlviewer`](https://github.com/cosmos/iavl/tree/master/cmd/iaviewer):

```bash
go run github.com/orijtech/chdbg ../application_math2.db ..
/application_v192.db 11317300 
chdbg: hash mismatch: 2381605E1374EE6F71F3F7BCECF6AE98B24C445A830C23B30424F1BE3E5DA7F4 != 28416BB9D35C766372F701B998B58927757B828E02E1D982167B7181D03C28EE
chdbg: key 2067616D6D2F706F6F6C2F3830362F7375706572756E626F6E64696E672F6F736D6F76616C6F706572316C3338373933636A6A6B73396B3730776B7864713777336175686A6D71753973396C346E6A792F6E6F64652F000000044C1FF2520000: value mismatch
chdbg: value a 
�R100000000000000
, value b 


L�R1000000000000000
```
- Debugger configuration for vscode exists in `.vscode/launch.json`

Debugging guide: https://app.clickup.com/37420681/v/dc/13nzm9-23133/13nzm9-46933

# LICENSE
Copyright Cosmos Network Authors. All Rights Reserved.
