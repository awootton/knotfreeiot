## BadJSON text recognizer.

The world has no shortage of pretty good text parsers but here's an idiosyncratic text chopper anyway. It will separate `GET /docs/index.html HTTP/1.1` into three byte arrays and also `["GET","/docs/index.html","HTTP/1.1"]` into the same three. It can parse `{"key1":"val1","key2":"val2"}` into 4 strings using a minimum of code.

For example, all the forms below parse into the same 4 byte arrays: 

`abc:def,ghi:jkl` and 

`"abc":"def","ghi":"jkl"`  and 

`abc,def,ghi,jkl"`  and

`abc def ghi "jkl`  and

`"abc":"def","ghi":"jkl"`  and the bizarre form

`"abc""def""ghi""jkl"` and also the common form 

`{"abc":"def","ghi":"jkl"}` we can also declare the bytes directly in hex or base64:

`$616263 $646566 =Z2hp =amts` also becomes the same three byte arrays (which is the true reason I wrote it).


TODO: Replace with something more formal.

