# some tests are old and some take too long to run
# fixme: they ALL should run.

go test -run=TestBills ./iot/test
go test -run=TestBillingAccumulatorContact ./iot/test

go test -run=TestReserveOneName ./iot/test
go test -run=TestGetProxyStatus ./iot/test
go test -run=TestGetA ./iot/test
go test -run=TestBills ./iot/test
go test -run=TestBills ./iot/test
go test -run=TestBills ./iot/test

go test -run=TestNameApiList ./iot/test

go test -run=TestNameApiDeleteName ./iot/test
go test -run=TestNameApiSetOption ./iot/test
go test -run=TestNameApiGetOption ./iot/test
go test -run=TestNameApiDetails ./iot/test
go test -run=TestNameApiAddName ./iot/test
go test -run=TestNameApiList ./iot/test
go test -run=TestUrl ./iot/test
go test -run=TestUrlFancy ./iot/test
go test -run=TestMapToString ./iot/test

go test -run=TestSubDomain ./iot/test
# go test -run=TestSomeApis ./iot/test









