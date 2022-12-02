# Conc-BST

Efficient, concurrent BST with minimal locking!

The goal of this repository is to allow multiple processes to safely:

* Write to the tree (altering tree structure)
* Iterate over the tree
* Read from the tree

We assume that the user has no race conditions on P1 writing to key 1, and P2 reading from key 2 simultaneously (though the code could be minimally adjusted to support this).
Instead, our work on race conditions is focused on making tree re-structuring for inserts & deletes safe.

This is based on [Practical Concurrent Traversals in Search Trees](https://files.sri.inf.ethz.ch/website/papers/ppopp18.pdf), cited in code as `[DVY18]`. Though that implementation works with data races

We extensively cite [Practical Concurrent Binary Search Trees via Logical Ordering](https://www.cs.technion.ac.il/~yahave/papers/ppopp14-trees.pdf) as `[DVY14]`.
