#!/usr/bin/env node
/**
 * extract.js — Parse TypeScript .d.ts (or .ts) files and output structured JSON
 * for the Go AID generator to consume.
 *
 * Usage: node extract.js <file.d.ts> [file2.d.ts ...]
 * Output: JSON array of module descriptors to stdout.
 */

const ts = require("typescript");
const fs = require("fs");
const path = require("path");

function extractFile(filePath) {
  const source = fs.readFileSync(filePath, "utf-8");
  const sourceFile = ts.createSourceFile(
    filePath,
    source,
    ts.ScriptTarget.Latest,
    true
  );

  const result = {
    file: filePath,
    module: path.basename(filePath, path.extname(filePath)).replace(/\.d$/, ""),
    functions: [],
    classes: [],
    interfaces: [],
    types: [],
    enums: [],
    constants: [],
  };

  ts.forEachChild(sourceFile, (node) => visitTopLevel(node, result, source));

  return result;
}

function visitTopLevel(node, result, source) {
  // Skip non-exported in .d.ts (everything is implicitly exported) or check export modifier
  const isExported =
    node.modifiers?.some((m) => m.kind === ts.SyntaxKind.ExportKeyword) ||
    // In .d.ts files, top-level declarations are implicitly exported
    true;

  if (ts.isFunctionDeclaration(node) && node.name) {
    result.functions.push(extractFunction(node));
  } else if (ts.isClassDeclaration(node) && node.name) {
    result.classes.push(extractClass(node));
  } else if (ts.isInterfaceDeclaration(node) && node.name) {
    result.interfaces.push(extractInterface(node));
  } else if (ts.isTypeAliasDeclaration(node) && node.name) {
    result.types.push(extractTypeAlias(node));
  } else if (ts.isEnumDeclaration(node) && node.name) {
    result.enums.push(extractEnum(node));
  } else if (ts.isVariableStatement(node)) {
    for (const decl of node.declarationList.declarations) {
      if (ts.isIdentifier(decl.name)) {
        result.constants.push(extractConstant(decl));
      }
    }
  } else if (ts.isModuleDeclaration(node)) {
    // namespace or module — recurse into body
    if (node.body && ts.isModuleBlock(node.body)) {
      for (const stmt of node.body.statements) {
        visitTopLevel(stmt, result, source);
      }
    }
  }
}

function extractFunction(node) {
  const fn = {
    name: node.name.text,
    async: node.modifiers?.some((m) => m.kind === ts.SyntaxKind.AsyncKeyword) || false,
    typeParams: extractTypeParams(node.typeParameters),
    params: extractParams(node.parameters),
    returnType: node.type ? typeToString(node.type) : "void",
    jsdoc: extractJSDoc(node),
  };
  return fn;
}

function extractClass(node) {
  const cls = {
    name: node.name.text,
    typeParams: extractTypeParams(node.typeParameters),
    extends: null,
    implements: [],
    members: [],
    jsdoc: extractJSDoc(node),
  };

  if (node.heritageClauses) {
    for (const clause of node.heritageClauses) {
      if (clause.token === ts.SyntaxKind.ExtendsKeyword) {
        cls.extends = clause.types.map((t) => typeToString(t)).join(", ");
      }
      if (clause.token === ts.SyntaxKind.ImplementsKeyword) {
        cls.implements = clause.types.map((t) => typeToString(t));
      }
    }
  }

  for (const member of node.members) {
    if (ts.isMethodDeclaration(member) || ts.isMethodSignature(member)) {
      const isStatic = member.modifiers?.some((m) => m.kind === ts.SyntaxKind.StaticKeyword);
      const isPrivate = member.modifiers?.some(
        (m) => m.kind === ts.SyntaxKind.PrivateKeyword
      );
      if (isPrivate) continue;

      cls.members.push({
        kind: "method",
        name: member.name?.getText?.() || member.name?.text || "",
        static: isStatic || false,
        async: member.modifiers?.some((m) => m.kind === ts.SyntaxKind.AsyncKeyword) || false,
        typeParams: extractTypeParams(member.typeParameters),
        params: extractParams(member.parameters),
        returnType: member.type ? typeToString(member.type) : "void",
        jsdoc: extractJSDoc(member),
      });
    } else if (ts.isPropertyDeclaration(member) || ts.isPropertySignature(member)) {
      const isPrivate = member.modifiers?.some(
        (m) => m.kind === ts.SyntaxKind.PrivateKeyword
      );
      if (isPrivate) continue;

      cls.members.push({
        kind: "property",
        name: member.name?.getText?.() || member.name?.text || "",
        type: member.type ? typeToString(member.type) : "any",
        readonly: member.modifiers?.some((m) => m.kind === ts.SyntaxKind.ReadonlyKeyword) || false,
        optional: !!member.questionToken,
        jsdoc: extractJSDoc(member),
      });
    } else if (ts.isConstructorDeclaration(member)) {
      cls.members.push({
        kind: "constructor",
        name: "constructor",
        params: extractParams(member.parameters),
        jsdoc: extractJSDoc(member),
      });
    }
  }

  return cls;
}

function extractInterface(node) {
  const iface = {
    name: node.name.text,
    typeParams: extractTypeParams(node.typeParameters),
    extends: [],
    members: [],
    jsdoc: extractJSDoc(node),
  };

  if (node.heritageClauses) {
    for (const clause of node.heritageClauses) {
      if (clause.token === ts.SyntaxKind.ExtendsKeyword) {
        iface.extends = clause.types.map((t) => typeToString(t));
      }
    }
  }

  for (const member of node.members) {
    if (ts.isMethodSignature(member)) {
      iface.members.push({
        kind: "method",
        name: member.name?.getText?.() || member.name?.text || "",
        typeParams: extractTypeParams(member.typeParameters),
        params: extractParams(member.parameters),
        returnType: member.type ? typeToString(member.type) : "void",
        optional: !!member.questionToken,
        jsdoc: extractJSDoc(member),
      });
    } else if (ts.isPropertySignature(member)) {
      iface.members.push({
        kind: "property",
        name: member.name?.getText?.() || member.name?.text || "",
        type: member.type ? typeToString(member.type) : "any",
        readonly: member.modifiers?.some((m) => m.kind === ts.SyntaxKind.ReadonlyKeyword) || false,
        optional: !!member.questionToken,
        jsdoc: extractJSDoc(member),
      });
    } else if (ts.isCallSignatureDeclaration(member)) {
      iface.members.push({
        kind: "call",
        params: extractParams(member.parameters),
        returnType: member.type ? typeToString(member.type) : "void",
      });
    } else if (ts.isIndexSignatureDeclaration(member)) {
      iface.members.push({
        kind: "index",
        params: extractParams(member.parameters),
        returnType: member.type ? typeToString(member.type) : "any",
      });
    }
  }

  return iface;
}

function extractTypeAlias(node) {
  return {
    name: node.name.text,
    typeParams: extractTypeParams(node.typeParameters),
    type: typeToString(node.type),
    jsdoc: extractJSDoc(node),
  };
}

function extractEnum(node) {
  return {
    name: node.name.text,
    members: node.members.map((m) => ({
      name: m.name?.getText?.() || m.name?.text || "",
      value: m.initializer ? m.initializer.getText?.() || "" : null,
    })),
    jsdoc: extractJSDoc(node),
  };
}

function extractConstant(node) {
  return {
    name: node.name.text,
    type: node.type ? typeToString(node.type) : "any",
    jsdoc: extractJSDoc(node.parent?.parent),
  };
}

function extractTypeParams(typeParams) {
  if (!typeParams) return [];
  return typeParams.map((tp) => {
    const param = { name: tp.name.text };
    if (tp.constraint) {
      param.constraint = typeToString(tp.constraint);
    }
    if (tp.default) {
      param.default = typeToString(tp.default);
    }
    return param;
  });
}

function extractParams(params) {
  if (!params) return [];
  return params.map((p) => {
    const param = {
      name: ts.isIdentifier(p.name) ? p.name.text : p.name.getText?.() || "...",
      type: p.type ? typeToString(p.type) : "any",
      optional: !!p.questionToken || !!p.initializer,
      rest: !!p.dotDotDotToken,
    };
    return param;
  });
}

function typeToString(typeNode) {
  if (!typeNode) return "any";

  // Use TypeScript's built-in printer for accurate representation
  const printer = ts.createPrinter({ removeComments: true });
  const sourceFile = typeNode.getSourceFile?.();
  if (sourceFile) {
    try {
      return printer.printNode(ts.EmitHint.Unspecified, typeNode, sourceFile);
    } catch {
      // Fallback
    }
  }

  // Manual fallback for simple cases
  if (ts.isTypeReferenceNode(typeNode)) {
    let name = typeNode.typeName?.getText?.() || typeNode.typeName?.text || "";
    if (typeNode.typeArguments) {
      name += "<" + typeNode.typeArguments.map(typeToString).join(", ") + ">";
    }
    return name;
  }
  if (typeNode.kind === ts.SyntaxKind.StringKeyword) return "string";
  if (typeNode.kind === ts.SyntaxKind.NumberKeyword) return "number";
  if (typeNode.kind === ts.SyntaxKind.BooleanKeyword) return "boolean";
  if (typeNode.kind === ts.SyntaxKind.VoidKeyword) return "void";
  if (typeNode.kind === ts.SyntaxKind.AnyKeyword) return "any";
  if (typeNode.kind === ts.SyntaxKind.NeverKeyword) return "never";
  if (typeNode.kind === ts.SyntaxKind.UndefinedKeyword) return "undefined";
  if (typeNode.kind === ts.SyntaxKind.NullKeyword) return "null";

  return "any";
}

function extractJSDoc(node) {
  if (!node) return "";
  // Check for JSDoc comments
  const jsDocs = ts.getJSDocCommentsAndTags?.(node) || node.jsDoc;
  if (jsDocs && jsDocs.length > 0) {
    const doc = jsDocs[0];
    if (doc.comment) {
      if (typeof doc.comment === "string") return doc.comment;
      // Handle structured comment
      return doc.comment.map((c) => c.text || "").join("");
    }
  }
  return "";
}

// Main
const files = process.argv.slice(2);
if (files.length === 0) {
  console.error("Usage: node extract.js <file.d.ts> [file2.d.ts ...]");
  process.exit(1);
}

const results = files.map(extractFile);
console.log(JSON.stringify(results, null, 2));
