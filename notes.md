# Notes on Script Embedding
_My summary of OrionSR's findings on embedding scripts in GTA:SA save files._

## Theory
While save files don't contain actual SCM code, we can still modify them to run our
own code. This is because a script can jump to an address that is not
actually a code address; in other words, you can jump to data, 
*and the game will interpret it as code*. Typically, doing this would just cause the
game to crash, because most variable data is not valid SCM code. Even if it was,
chances are that it would do something random and cause a crash that way. However,
if we can make it so that the variable data *is* valid code, we can run that code.

This method requires us to have access to variable data. Luckily, the entire global
variable store is saved to, and loaded from, save files. Thus, we can change
variables inside the global store, and as long as we can find a way to jump to the
offset of our changed variables, we can run that data as code.