# nat : nietzsche archive transpiler

Or, making the eKGWB greppable.

To get the full [KGW in Markdown][kgw], with emphasis. The [HKA][hka] I have is plaintext with no emphasis markers.

## Citations

The problem is matching the citations.

the eKGWB uses some bullshit NF-YEAR,00[0] syntax. Which doesn't directly match any of the schemes in the HKA nor the KGW itself.

so this:

    eKGWB/NF-1888,15[1]

must map to this:

    Aphorism n=12381 id='VIII.15[1]' kgw='VIII-3.193' ksa='13.401'

which means we have to know that 1888,15 maps to VIII.15, without a 1:1 relationship between 1888 and VIII.

Luckily, the eKGWB does include the titles of Nietzsche's notebooks as assigned by the archive. So we go look for something like this:

    [15 = W II 6a. Frühjahr 1888]

and match it against a similar string in the HKA. Then we map all aphorisms following that string in the HKA, and assign them to the corresponding entries in the eKGWB. But, there seems to be errors and discrepancies - mostly in the eKGWB it seems. We get around it by looking for the minimal unique key, and make a guess.

[kgw]: https://github.com/brtholomy/nat/tree/master/output
[hka]: https://raw.githubusercontent.com/brtholomy/nat/refs/heads/master/sources/HKA.txt
