import Editor from '@monaco-editor/react';

type DocumentMonacoEditorProps = {
  value: string;
  onChange?: (value: string) => void;
  readOnly?: boolean;
  height?: string;
};

export default function DocumentMonacoEditor({
  value,
  onChange,
  readOnly = false,
  height = '420px',
}: DocumentMonacoEditorProps) {
  return (
    <div className="editor-monaco">
      <Editor
        height={height}
        defaultLanguage="plaintext"
        value={value}
        onChange={readOnly ? undefined : (nextValue) => onChange?.(nextValue ?? '')}
        options={{
          readOnly,
          minimap: { enabled: false },
          wordWrap: 'on',
          scrollBeyondLastLine: false,
          fontSize: 14,
          lineNumbers: 'on',
          automaticLayout: true,
          padding: { top: 12, bottom: 12 },
        }}
      />
    </div>
  );
}
