
GoLang Counter Utility
=====================

*Why a counter utility?*

When I programmed in C, we'd routinely add a counter to each place in
the code where there was a control-flow affecting test.  Then we had
commands to print out the counters and since our code was able to run
for a long time without crashing, we'd get a sort of free profile of
the application flow.  


*Is it good to use?*

I'm using it.  

*What is it? *

This code offers an "increment a counter" API that is non-blocking,
using only channels not locking, and which requires no upfront
configuration (you provide a string name of the counter).  Each minute
the counts will be printed out (or if you provide a logging API that
conforms to the given interface, logged TBD).  The numbers will be
old-school aligned and tabularized.  If you provide a callback, then
each minute you'll get a callback with all string names of the
existing counters and values for sending to a TSDB type system for
aggregating these metrics.

*Who owns this code?*

Chris Lane

*Adivce for starting out*

If you integrate, please let me or them know of your experience and
any suggestions for improvement.

The current API can best be seen in the _test files probably.  

One thing to be aware of is that calling counter_init() will start up
a go routine or 2 to manage per minute processing.  There will be an API to
kill them if you are doing odd stuff but normally they will for your
process life.  

*Requirements*

None at present.  
