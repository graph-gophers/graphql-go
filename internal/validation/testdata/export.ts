import * as fs from 'node:fs';
import { createHash } from 'crypto';
import { printSchema } from 'graphql/src/utilities/printSchema.js';
import { schemas, testCases } from 'graphql/src/validation/__tests__/harness.js';

// TODO: Fix test failures.
// require('graphql/src/validation/__tests__/ExecutableDefinitions-test');
import 'graphql/src/validation/__tests__/FieldsOnCorrectTypeRule-test.js';
import 'graphql/src/validation/__tests__/FragmentsOnCompositeTypesRule-test.js';
import 'graphql/src/validation/__tests__/KnownArgumentNamesRule-test.js';
import 'graphql/src/validation/__tests__/KnownDirectivesRule-test.js';
import 'graphql/src/validation/__tests__/KnownFragmentNamesRule-test.js';
import 'graphql/src/validation/__tests__/KnownTypeNamesRule-test.js';
import 'graphql/src/validation/__tests__/LoneAnonymousOperationRule-test.js';
import 'graphql/src/validation/__tests__/NoFragmentCyclesRule-test.js';
import 'graphql/src/validation/__tests__/NoUndefinedVariablesRule-test.js';
import 'graphql/src/validation/__tests__/NoUnusedFragmentsRule-test.js';
import 'graphql/src/validation/__tests__/NoUnusedVariablesRule-test.js';
import 'graphql/src/validation/__tests__/OverlappingFieldsCanBeMergedRule-test.js';
import 'graphql/src/validation/__tests__/PossibleFragmentSpreadsRule-test.js';
import 'graphql/src/validation/__tests__/ProvidedRequiredArgumentsRule-test.js';
import 'graphql/src/validation/__tests__/ScalarLeafsRule-test.js';
// TODO: Add support for subscriptions.
// require('graphql/src/validation/__tests__/SingleFieldSubscriptions-test.js');
import 'graphql/src/validation/__tests__/UniqueArgumentNamesRule-test.js';
import 'graphql/src/validation/__tests__/UniqueDirectivesPerLocationRule-test.js';
import 'graphql/src/validation/__tests__/UniqueFragmentNamesRule-test.js';
import 'graphql/src/validation/__tests__/UniqueInputFieldNamesRule-test.js';
import 'graphql/src/validation/__tests__/UniqueOperationNamesRule-test.js';
import 'graphql/src/validation/__tests__/UniqueVariableNamesRule-test.js';
// TODO: Fix test failures.
// require('graphql/src/validation/__tests__/ValuesofCorrectType-test');
import 'graphql/src/validation/__tests__/VariablesAreInputTypesRule-test.js';
// TODO: Fix test failures.
// require('graphql/src/validation/__tests__/VariablesDefaultValueAllowed-test');
import 'graphql/src/validation/__tests__/VariablesInAllowedPositionRule-test.js';

// Schema index in the source array can be unstable, as its dependent on the order they are used in the registered test
// files. The SHA256 of the schema will change if there are any changes to the content, but is a better reference than
// the schema indexes all changing when a new schema is inserted.
let s = schemas().map(s => {
  const sdl = printSchema(s)
  const id = createHash('sha256').update(sdl).digest('base64');
  const v: { id: string; sdl: string; } = {id: id, sdl: sdl};
  return v;
});

const tests = testCases().map(c => {
  const schema = s[c.schema];
  return {
    name: c.name,
	  rule: c.rule,
	  schema: schema.id,
	  query: c.query,
	  errors: c.errors,
  }
});

// Order based on the schema string to provide semi-stable ordering
s = s.sort((a, b) => a.sdl.localeCompare(b.sdl));

let output = JSON.stringify({schemas: s, tests: tests}, null, 2)
output = output.replace(/ Did you mean to use an inline fragment on [^?]*\?/g, '');
// Ignore suggested types in errors, which graphql-go does not currently produce.
output = output.replace(/ Did you mean \\"[A-Z].*\"\?/g, '');
fs.writeFileSync("tests.json", output);
