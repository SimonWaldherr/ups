# UPS - Uncommon Printing System

[![Build Status](https://travis-ci.org/SimonWaldherr/ups.svg?branch=master)](https://travis-ci.org/SimonWaldherr/ups)  
[![GoDoc](https://godoc.org/github.com/SimonWaldherr/ups?status.svg)](https://godoc.org/github.com/SimonWaldherr/ups)  

I wrote the [Uncommon Printing System](https://simonwaldherr.de/go/ups) a long time ago to replace a proprietary printing system called NiceWatch by [NiceLabel](https://www.nicelabel.com). 
Don't get me wrong, [Nice Label Designer](https://www.nicelabel.com/design-and-print) is still the best WYSIWYG Label Editor on Earth, but NiceWatch is slow and unstable. 
The UPS is programmed to support Label Templates designed with NiceLabel Designer and print them on ZPL compatible printers. 

This repository contains a refactored version of **UPS**. It only contains general purpose features, features like:

* handling invalid XML-Files from SAP systems
* reload missing data in XML-Files from SAP systems
* save log-data to BI (business intelligence) system
* transfer material master data to a sub-system
* load printer data from a SAP database table

I use UPS in a customized version to print up to 10000 labels daily.
UPS can also do a lot more with ease, but in the current case of application it is not needed.
You can even run UPS on a Raspberry Pi.

Currently I only print on Zebra ZM400, Zebra ZT410, Zebra QL 420 (plus) and Zebra QLn 420, but I plan to extend the UPS to support non ZPL printers as well.

## Test it

to test the application you can simply follow these steps:

1. ```go get simonwaldherr.de/go/ups/cmd/ups```
1. ```ups &```
1. ```nc -l 9100 > zpl &```
1. ```cat xmlreq.xml | nc localhost 30000```
1. ```kill -9 $(pidof ups)```

## Why

Why what? Why I wrote the application? Because I did not want to bother anymore with an unstable proprietary software! 
I wasted several hours a week managing the NiceWatch system and keeping it running. 
I had much better things to do and that's why I wrote UPS.  

Why I made some choices the way I made them? Mostly there is also a reasonable cause - you want an example? 
There is a function called ```cdatafy``` which adds ```CDATA```-sections to the XML-string. 
You may ask why on earth someone would need such a function. 
Because many SAP developers don't know anything about XML in general and XML marshalling in specific, 
so they just concatenate strings and as result they create invalid XMLs. 
It seems like the favorite word of most SAP consultants is standard, 
but when it comes to actual W3C, WHATWG, IETF, ISO, … standards, they do not care.

## Is it any good?

[Yes](https://news.ycombinator.com/item?id=3067434)

## License

I have not decided yet
