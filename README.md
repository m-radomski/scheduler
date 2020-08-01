# scheduler
`scheduler` is a simple command line interface for planning your route using premade JSON databases.
It's created for [Cracow's](http://ztp.krakow.pl/) public transport system, but anybody with the correct database can use it.

# Installation
For people that know what they are going, all you need is to have `go` installed on your computer.
From there you can type the following command:

```
go get -u https://github.com/m-radomski/scheduler
```

This downloads and installs the program into your `GOPATH`, from there just run the command `scheduler`. 

# Usage
The window is split into 3 main panels.
The biggest one shows you relevant connections, which line number it's on, stop name and next departure time.
In the bottom left you can search for connections between stops, while the bottom right allows for a fuzzy search by line number or stop name.

To move around inside a focused window use the `Tab` key, your mouse or the arrow keys, for confirming use the `Enter` key.
To switch focus between windows on the Search Page press `Ctrl + Space`.
When you are in the schedule of a certain line on a certain stop you can go to the next or previous stops schedule by pressing `Ctrl + N` for *Next* or `Ctrl + P` for *Previous*.
If you wish to update your database you can press `Ctrl + R`, it will download the lastest version of the schedule from the web.

Once you searched for something, you are now controlling the connections list.
Using the `Enter` key on one shows you the schedule for that particular stop, line and it's direction.
To go back to searching again press `Esc`.
