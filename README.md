# scheduler
`scheduler` is a simple command line interface for planning your route using premade JSON databases.
It's created for [Cracow's](http://ztp.krakow.pl/) public transport system, but anybody with the correct database can use it.

# Usage
The window is split into 3 main panels.
The biggest one shows you relevant connections, which line number it's on, stop name and next departure time.
In the bottom left you can search for connections between stops, while the bottom right allows for a fuzzy search by line number or stop name.

To move around use the `Tab` key as well as your mouse, for confirming use the `Enter` key.

Once you searched for something, you are now controlling the connections list.
Using the `Enter` key on one shows you the schedule for that particular stop, line and it's direction.
To go back to searching again press `Esc`.
