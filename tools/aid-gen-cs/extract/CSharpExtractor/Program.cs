using System.Text.Json;
using Microsoft.CodeAnalysis;
using Microsoft.CodeAnalysis.CSharp;
using Microsoft.CodeAnalysis.CSharp.Syntax;

/// <summary>
/// Parses C# source files using Roslyn and outputs structured JSON
/// for the Go AID generator to consume.
/// Usage: dotnet run -- file1.cs file2.cs ...
/// </summary>

if (args.Length == 0)
{
    Console.Error.WriteLine("Usage: CSharpExtractor <file.cs> [file2.cs ...]");
    return 1;
}

var results = new List<object>();

foreach (var filePath in args)
{
    if (!File.Exists(filePath))
    {
        Console.Error.WriteLine($"File not found: {filePath}");
        continue;
    }

    var source = File.ReadAllText(filePath);
    var tree = CSharpSyntaxTree.ParseText(source, path: filePath);
    var root = tree.GetCompilationUnitRoot();

    var extractor = new FileExtractor(filePath);
    extractor.Visit(root);
    results.Add(extractor.GetResult());
}

var json = JsonSerializer.Serialize(results, new JsonSerializerOptions
{
    WriteIndented = true,
    PropertyNamingPolicy = JsonNamingPolicy.CamelCase,
    DefaultIgnoreCondition = System.Text.Json.Serialization.JsonIgnoreCondition.WhenWritingNull
});
Console.WriteLine(json);
return 0;

class FileExtractor : CSharpSyntaxWalker
{
    private readonly string _filePath;
    private readonly List<object> _classes = new();
    private readonly List<object> _interfaces = new();
    private readonly List<object> _structs = new();
    private readonly List<object> _enums = new();
    private readonly List<object> _delegates = new();
    private string _namespace = "";

    public FileExtractor(string filePath) : base(SyntaxWalkerDepth.Node)
    {
        _filePath = filePath;
    }

    public object GetResult() => new
    {
        File = _filePath,
        Module = Path.GetFileNameWithoutExtension(_filePath),
        Namespace = _namespace,
        Classes = _classes,
        Interfaces = _interfaces,
        Structs = _structs,
        Enums = _enums,
        Delegates = _delegates
    };

    public override void VisitNamespaceDeclaration(NamespaceDeclarationSyntax node)
    {
        _namespace = node.Name.ToString();
        base.VisitNamespaceDeclaration(node);
    }

    public override void VisitFileScopedNamespaceDeclaration(FileScopedNamespaceDeclarationSyntax node)
    {
        _namespace = node.Name.ToString();
        base.VisitFileScopedNamespaceDeclaration(node);
    }

    public override void VisitClassDeclaration(ClassDeclarationSyntax node)
    {
        if (!IsPublic(node.Modifiers)) return;

        var members = ExtractMembers(node.Members);
        _classes.Add(new
        {
            Name = node.Identifier.Text,
            TypeParams = ExtractTypeParams(node.TypeParameterList),
            BaseTypes = node.BaseList?.Types.Select(t => t.ToString()).ToList(),
            IsAbstract = node.Modifiers.Any(SyntaxKind.AbstractKeyword),
            IsStatic = node.Modifiers.Any(SyntaxKind.StaticKeyword),
            IsSealed = node.Modifiers.Any(SyntaxKind.SealedKeyword),
            Members = members,
            Doc = GetXmlDoc(node)
        });
    }

    public override void VisitStructDeclaration(StructDeclarationSyntax node)
    {
        if (!IsPublic(node.Modifiers)) return;

        var members = ExtractMembers(node.Members);
        _structs.Add(new
        {
            Name = node.Identifier.Text,
            TypeParams = ExtractTypeParams(node.TypeParameterList),
            BaseTypes = node.BaseList?.Types.Select(t => t.ToString()).ToList(),
            Members = members,
            Doc = GetXmlDoc(node)
        });
    }

    public override void VisitInterfaceDeclaration(InterfaceDeclarationSyntax node)
    {
        if (!IsPublic(node.Modifiers)) return;

        var members = ExtractMembers(node.Members);
        _interfaces.Add(new
        {
            Name = node.Identifier.Text,
            TypeParams = ExtractTypeParams(node.TypeParameterList),
            BaseTypes = node.BaseList?.Types.Select(t => t.ToString()).ToList(),
            Members = members,
            Doc = GetXmlDoc(node)
        });
    }

    public override void VisitEnumDeclaration(EnumDeclarationSyntax node)
    {
        if (!IsPublic(node.Modifiers)) return;

        _enums.Add(new
        {
            Name = node.Identifier.Text,
            Members = node.Members.Select(m => new
            {
                Name = m.Identifier.Text,
                Value = m.EqualsValue?.Value.ToString(),
                Doc = GetXmlDoc(m)
            }).ToList(),
            Doc = GetXmlDoc(node)
        });
    }

    public override void VisitDelegateDeclaration(DelegateDeclarationSyntax node)
    {
        if (!IsPublic(node.Modifiers)) return;

        _delegates.Add(new
        {
            Name = node.Identifier.Text,
            TypeParams = ExtractTypeParams(node.TypeParameterList),
            Params = ExtractParams(node.ParameterList),
            ReturnType = node.ReturnType.ToString(),
            Doc = GetXmlDoc(node)
        });
    }

    private List<object> ExtractMembers(SyntaxList<MemberDeclarationSyntax> members)
    {
        var result = new List<object>();

        foreach (var member in members)
        {
            switch (member)
            {
                case MethodDeclarationSyntax method when IsPublicOrProtected(method.Modifiers):
                    result.Add(new
                    {
                        Kind = "method",
                        Name = method.Identifier.Text,
                        IsStatic = method.Modifiers.Any(SyntaxKind.StaticKeyword),
                        IsAsync = method.Modifiers.Any(SyntaxKind.AsyncKeyword),
                        IsAbstract = method.Modifiers.Any(SyntaxKind.AbstractKeyword),
                        IsVirtual = method.Modifiers.Any(SyntaxKind.VirtualKeyword),
                        TypeParams = ExtractTypeParams(method.TypeParameterList),
                        Params = ExtractParams(method.ParameterList),
                        ReturnType = method.ReturnType.ToString(),
                        Doc = GetXmlDoc(method)
                    });
                    break;

                case PropertyDeclarationSyntax prop when IsPublicOrProtected(prop.Modifiers):
                    result.Add(new
                    {
                        Kind = "property",
                        Name = prop.Identifier.Text,
                        Type = prop.Type.ToString(),
                        IsStatic = prop.Modifiers.Any(SyntaxKind.StaticKeyword),
                        HasGetter = prop.AccessorList?.Accessors.Any(a => a.IsKind(SyntaxKind.GetAccessorDeclaration)) ?? prop.ExpressionBody != null,
                        HasSetter = prop.AccessorList?.Accessors.Any(a => a.IsKind(SyntaxKind.SetAccessorDeclaration) || a.IsKind(SyntaxKind.InitAccessorDeclaration)) ?? false,
                        Doc = GetXmlDoc(prop)
                    });
                    break;

                case FieldDeclarationSyntax field when IsPublicOrProtected(field.Modifiers):
                    foreach (var variable in field.Declaration.Variables)
                    {
                        result.Add(new
                        {
                            Kind = "field",
                            Name = variable.Identifier.Text,
                            Type = field.Declaration.Type.ToString(),
                            IsStatic = field.Modifiers.Any(SyntaxKind.StaticKeyword),
                            IsReadonly = field.Modifiers.Any(SyntaxKind.ReadOnlyKeyword),
                            IsConst = field.Modifiers.Any(SyntaxKind.ConstKeyword),
                            Value = variable.Initializer?.Value.ToString(),
                            Doc = GetXmlDoc(field)
                        });
                    }
                    break;

                case ConstructorDeclarationSyntax ctor when IsPublicOrProtected(ctor.Modifiers):
                    result.Add(new
                    {
                        Kind = "constructor",
                        Name = ctor.Identifier.Text,
                        Params = ExtractParams(ctor.ParameterList),
                        Doc = GetXmlDoc(ctor)
                    });
                    break;

                case EventDeclarationSyntax evt when IsPublicOrProtected(evt.Modifiers):
                    result.Add(new
                    {
                        Kind = "event",
                        Name = evt.Identifier.Text,
                        Type = evt.Type.ToString(),
                        Doc = GetXmlDoc(evt)
                    });
                    break;

                case IndexerDeclarationSyntax indexer when IsPublicOrProtected(indexer.Modifiers):
                    result.Add(new
                    {
                        Kind = "indexer",
                        Params = ExtractParams(indexer.ParameterList),
                        ReturnType = indexer.Type.ToString(),
                        Doc = GetXmlDoc(indexer)
                    });
                    break;
            }
        }

        return result;
    }

    private static List<object> ExtractParams(BaseParameterListSyntax? paramList)
    {
        if (paramList == null) return new();
        return paramList.Parameters.Select(p => (object)new
        {
            Name = p.Identifier.Text,
            Type = p.Type?.ToString() ?? "object",
            IsOptional = p.Default != null,
            IsParams = p.Modifiers.Any(SyntaxKind.ParamsKeyword),
            IsRef = p.Modifiers.Any(SyntaxKind.RefKeyword),
            IsOut = p.Modifiers.Any(SyntaxKind.OutKeyword),
            Default = p.Default?.Value.ToString()
        }).ToList();
    }

    private static List<object>? ExtractTypeParams(TypeParameterListSyntax? typeParams)
    {
        if (typeParams == null) return null;
        return typeParams.Parameters.Select(tp => (object)new
        {
            Name = tp.Identifier.Text
        }).ToList();
    }

    private static bool IsPublic(SyntaxTokenList modifiers) =>
        modifiers.Any(SyntaxKind.PublicKeyword);

    private static bool IsPublicOrProtected(SyntaxTokenList modifiers) =>
        modifiers.Any(SyntaxKind.PublicKeyword) || modifiers.Any(SyntaxKind.ProtectedKeyword);

    private static string? GetXmlDoc(SyntaxNode node)
    {
        var trivia = node.GetLeadingTrivia()
            .Where(t => t.IsKind(SyntaxKind.SingleLineDocumentationCommentTrivia) ||
                       t.IsKind(SyntaxKind.MultiLineDocumentationCommentTrivia))
            .FirstOrDefault();

        if (trivia == default) return null;

        var structure = trivia.GetStructure();
        if (structure is DocumentationCommentTriviaSyntax doc)
        {
            var summary = doc.ChildNodes()
                .OfType<XmlElementSyntax>()
                .FirstOrDefault(e => e.StartTag.Name.ToString() == "summary");

            if (summary != null)
            {
                var text = string.Join(" ", summary.Content
                    .Select(c => c.ToString().Trim()))
                    .Replace("///", "")
                    .Trim();
                return string.IsNullOrWhiteSpace(text) ? null : text;
            }
        }

        return null;
    }
}
