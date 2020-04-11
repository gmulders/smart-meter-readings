#!/bin/bash

for CMD in `ls cmd`; do
  go build ./cmd/$CMD
done
