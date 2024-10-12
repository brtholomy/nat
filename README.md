# eKGWB

To get the full KGW in Markdown, with emphasis. The HKA I have is plaintext with no emphasis markers.

## Numbering

Problem is the numbering.

the eKGWB uses some bullshit NF-YEAR,00[0] numbering. Which doesn't directly match any of the schemes in the HKA.

so this:

    eKGWB/NF-1888,15[1]

must map to this:

    Aphorism n=12381 id='VIII.15[1]' kgw='VIII-3.193' ksa='13.401'

which means I have to somehow know that 1888,15 maps to VIII.15, but that:

    #eKGWB/NF-1885,45[7]

maps to:

    Aphorism n=10634 id='VII.45[7]' kgw='VII-3.452' ksa='11.710'

## pandoc

    pandoc --wrap=none --to=markdown-smart output.html -o test.md
