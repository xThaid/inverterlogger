# inverterlogger

A small library that encodes and decodes data from Solarman data loggers.
It works with wifi kits with a s/n starting with 17xxxxxxxx.

Please be careful with changing constants, as you can accidentialy send different command. 
So instead of fetching real-time data, you could change some setting of an inverter (this can be really dangerous).

Refer to the [example](https://github.com/xThaid/inverterlogger/example) to see how it can be used.

Hope this helps you to understand how the protocol works.