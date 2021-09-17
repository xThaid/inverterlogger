# Example for inverter logger

This example shows how this module can be used. It queries a logger for some registers and extracts useful information from it.

Please note that the register mapping is dependent on the inverter you have. Mine is Deye SUN-5K-G03 and registers with interesting data were found by trial and error.

To run this example firstly provide the inverter's IP address and SN in the `main.go`. Then execute:

```shell
go run .
```

The logger often timeouts and I don't know why. In this case just send the request again.