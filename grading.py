#!/usr/bin/env python

import os, time, sys
from os.path import isfile, join
import shutil

os.system('./build')

test_output = 'test_output'
tests = 'grading_tests'
output = 'results_1'
if len(sys.argv) == 2:
    tests = sys.argv[1]
try:
    shutil.rmtree(test_output)
except:
    pass
os.mkdir(test_output)
list_of_tests = [];
for f in os.listdir(tests):
	list_of_tests.append(f)
list_of_tests.sort()

print list_of_tests
output_file = open(output, 'w')

for f in list_of_tests:
    abs_f = join(tests, f)
    if isfile(abs_f):
        if f[len(f) - len('.input'):] == '.input':
            fn = f[:len(f) - len('.input')]
            output_file.write(fn + ' ')
            print fn,
            os.system('./master.py < ' + abs_f + \
                      ' 2> ' + join(test_output, fn+'.err') + \
                      ' > ' + join(test_output, fn+'.output'))

            with open(join(test_output, fn+'.output')) as fi:
                out = fi.read()
            with open(join(tests, fn+'.output')) as fi:
                std = fi.read()
            if out.strip() == std.strip():
            	output_file.write('correct \n')
                print 'correct'
            else:
            	output_file.write('wrong \n')
                print 'wrong'
            time.sleep(2)
