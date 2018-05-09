blwipe
=======

**blwipe** is a tool that performs *cryptographic erasure* on BitLocker volumes.
This allows you to "erase" BitLocker volumes very quickly by just rewriting 
areas where the key material is stored on the volume.

The tool is written in Go, so it should compile into a "static" binary and 
is easily cross-compilable.

Installation
=============

You will need to install [Go](https://golang.org/).

To download and compile *blwipe*, use `go get`:

	go get https://github.com/geekman/blwipe


Usage
======

Run *blwipe* on the BitLocker volume image to display details about the volume:

	blwipe /dev/sda1

It should tell you the location of the metadata blocks.
If you want it to dump the parsed structures, pass `-v`.

You can also use the entire device (e.g. `/dev/sda`) or image file and specify
an offset to use, for example `-offset 0x10000` if the partition starts there.
The specified offset can be of other bases, as long as you specify the right 
prefix (i.e. `0` for octal, `0x` for hex).

To actually wipe the volume, pass the `-wipe` flag:

	blwipe -wipe /dev/sda1

You will NOT receive any prompts or confirmation.


License
========

**blwipe is licensed under the 3-clause ("modified") BSD License.**

Copyright (C) 2018 Darell Tan

Redistribution and use in source and binary forms, with or without
modification, are permitted provided that the following conditions
are met:

1. Redistributions of source code must retain the above copyright
   notice, this list of conditions and the following disclaimer.
2. Redistributions in binary form must reproduce the above copyright
   notice, this list of conditions and the following disclaimer in the
   documentation and/or other materials provided with the distribution.
3. The name of the author may not be used to endorse or promote products
   derived from this software without specific prior written permission.

THIS SOFTWARE IS PROVIDED BY THE AUTHOR "AS IS" AND ANY EXPRESS OR
IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED WARRANTIES
OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE DISCLAIMED.
IN NO EVENT SHALL THE AUTHOR BE LIABLE FOR ANY DIRECT, INDIRECT,
INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT
NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE,
DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY
THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
(INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF
THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.

