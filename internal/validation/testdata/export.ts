import * as fs from 'node:fs';
import { printSchema } from 'graphql/src/utilities/printSchema.js';
import { schemas, testCases } from 'graphql/src/validation/__tests__/harness.js';

// TODO: Fix test failures.
// require('graphql/src/validation/__tests__/ExecutableDefinitions-test');
import 'graphql/src/validation/__tests__/FieldsOnCorrectTypeRule-test.js';
import 'graphql/src/validation/__tests__/FragmentsOnCompositeTypesRule-test.js';
// TODO: Fix test failures.
// require('graphql/src/validation/__tests__/KnownArgumentNames-test');
import 'graphql/src/validation/__tests__/KnownDirectivesRule-test.js';
import 'graphql/src/validation/__tests__/KnownFragmentNamesRule-test.js';
import 'graphql/src/validation/__tests__/KnownTypeNamesRule-test.js';
import 'graphql/src/validation/__tests__/LoneAnonymousOperationRule-test.js';
import 'graphql/src/validation/__tests__/NoFragmentCyclesRule-test.js';
import 'graphql/src/validation/__tests__/NoUndefinedVariablesRule-test.js';
import 'graphql/src/validation/__tests__/NoUnusedFragmentsRule-test.js';
import 'graphql/src/validation/__tests__/NoUnusedVariablesRule-test.js';
import 'graphql/src/validation/__tests__/OverlappingFieldsCanBeMergedRule-test.js';
// TODO: Fix test failures.
// require('graphql/src/validation/__tests__/PossibleFragmentSpreads-test');
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

let output = JSON.stringify({
	schemas: schemas().map(s => printSchema(s)),
	tests: testCases(),
}, null, 2)
output = output.replace(/ Did you mean to use an inline fragment on [^?]*\?/g, '');
// Ignore suggested types in errors, which graphql-go does not currently produce.
output = output.replace(/ Did you mean \\"[A-Z].*\"\?/g, '');
fs.writeFileSync("tests.json", output);
