const fs = require('node:fs');
const path = require('node:path');
const { createHash } = require('node:crypto');
const Module = require('node:module');
const ts = require('typescript');

Module._extensions['.ts'] = function compileTypeScript(module, filename) {
  const source = fs.readFileSync(filename, 'utf8');
  const compiled = ts.transpileModule(source, {
    compilerOptions: {
      module: ts.ModuleKind.CommonJS,
      moduleResolution: ts.ModuleResolutionKind.Node10,
      target: ts.ScriptTarget.ES2022,
      esModuleInterop: true,
      sourceMap: false,
      inlineSourceMap: false,
    },
    fileName: filename,
  });

  module._compile(compiled.outputText, filename);
};

const originalResolveFilename = Module._resolveFilename;
Module._resolveFilename = function patchedResolveFilename(request, parent, ...rest) {
  try {
    return originalResolveFilename.call(this, request, parent, ...rest);
  } catch (error) {
    if (
      error &&
      error.code === 'MODULE_NOT_FOUND' &&
      typeof request === 'string' &&
      request.endsWith('.js') &&
      parent &&
      typeof parent.filename === 'string' &&
      parent.filename.includes(`${path.sep}node_modules${path.sep}graphql${path.sep}src${path.sep}`)
    ) {
      const tsRequest = request.slice(0, -3) + '.ts';
      return originalResolveFilename.call(this, tsRequest, parent, ...rest);
    }

    throw error;
  }
};

const captured = [];
const suiteNames = [];
const schemaRefs = [];
const graphqlRootDir = path.dirname(require.resolve('graphql/package.json'));

const { printSchema } = require('graphql/src/utilities/printSchema.ts');

function describe(name, fn) {
  if (name === 'within schema language') {
    return;
  }

  suiteNames.push(name);
  try {
    fn();
  } finally {
    suiteNames.pop();
  }
}

describe.only = describe;
describe.skip = () => {};

function it(name, fn) {
  suiteNames.push(name);

  try {
    if (typeof fn === 'function') {
      if (fn.length > 0) {
        fn(() => {});
      } else {
        fn();
      }
    }
  } finally {
    suiteNames.pop();
  }
}

it.only = it;
it.skip = () => {};

const mocha = require('mocha');
mocha.describe = describe;
mocha.it = it;

global.describe = describe;
global.it = it;

function registerSchema(schema) {
  for (let i = 0; i < schemaRefs.length; i += 1) {
    if (schemaRefs[i] === schema) {
      return i;
    }
  }

  schemaRefs.push(schema);
  return schemaRefs.length - 1;
}

const harness = require('graphql/src/validation/__tests__/harness.ts');

harness.describe = describe;
harness.it = it;
harness.expectValidationErrorsWithSchema = function expectValidationErrorsWithSchema(
  schema,
  rule,
  queryStr,
) {
  return {
    toDeepEqual(errors) {
      captured.push({
        name: suiteNames.join('/'),
        rule: rule.name,
        schema: registerSchema(schema),
        query: queryStr,
        errors,
      });
    },
  };
};

harness.expectValidationErrors = function expectValidationErrors(rule, queryStr) {
  return harness.expectValidationErrorsWithSchema(harness.testSchema, rule, queryStr);
};

const validationTestsDir = path.join(graphqlRootDir, 'src/validation/__tests__');
const incompatibleSuites = new Set([
  // graphql-go does not expose graphql-js custom validation rule MaxIntrospectionDepthRule.
  'MaxIntrospectionDepthRule-test.ts',
  // graphql-go does not expose graphql-js custom validation rule NoSchemaIntrospectionCustomRule.
  'NoSchemaIntrospectionCustomRule-test.ts',
]);

const suiteFiles = fs
  .readdirSync(validationTestsDir, { withFileTypes: true })
  .filter(entry => entry.isFile() && entry.name.endsWith('-test.ts'))
  .map(entry => entry.name)
  .sort();

for (const suiteFile of suiteFiles) {
  if (incompatibleSuites.has(suiteFile)) {
    continue;
  }

  require(path.join(validationTestsDir, suiteFile));
}

let schemas = schemaRefs.map(schema => {
  const sdl = printSchema(schema);
  const id = createHash('sha256').update(sdl).digest('base64');
  return { id, sdl };
});

const tests = captured.map(testCase => {
  const schema = schemas[testCase.schema];
  return {
    name: testCase.name,
    rule: testCase.rule,
    schema: schema.id,
    query: testCase.query,
    errors: testCase.errors,
  };
});

schemas = schemas.sort((a, b) => a.sdl.localeCompare(b.sdl));

let output = JSON.stringify({ schemas, tests }, null, 2);
output = output.replace(/ Did you mean to use an inline fragment on [^?]*\?/g, '');
// Ignore suggested types in errors, which graphql-go does not currently produce.
output = output.replace(/ Did you mean \\"[A-Z].*\"\?/g, '');
fs.writeFileSync('tests.json', output);
console.log('Generated tests.json successfully');
