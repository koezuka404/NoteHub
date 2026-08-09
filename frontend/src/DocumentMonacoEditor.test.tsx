import { cleanup, fireEvent, render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { afterEach, describe, expect, it, vi } from 'vitest';
import DocumentMonacoEditor from './DocumentMonacoEditor';

describe('DocumentMonacoEditor', () => {
  afterEach(() => {
    cleanup();
  });

  it('calls onChange when editable', async () => {
    const user = userEvent.setup();
    const onChange = vi.fn();

    render(<DocumentMonacoEditor value="hello" onChange={onChange} />);

    await user.type(screen.getByTestId('monaco-editor'), '!');
    expect(onChange).toHaveBeenCalled();
  });

  it('does not call onChange when read only', async () => {
    const user = userEvent.setup();
    const onChange = vi.fn();

    render(<DocumentMonacoEditor value="hello" readOnly onChange={onChange} />);

    const editor = screen.getByTestId('monaco-editor') as HTMLTextAreaElement;
    expect(editor.readOnly).toBe(true);
    await user.type(editor, '!');
    expect(onChange).not.toHaveBeenCalled();
  });

  it('uses custom height', () => {
    render(<DocumentMonacoEditor value="hello" height="600px" />);
    expect(screen.getByTestId('monaco-editor')).toBeInTheDocument();
  });

  it('normalizes undefined editor values', () => {
    const onChange = vi.fn();
    render(<DocumentMonacoEditor value="hello" onChange={onChange} />);
    fireEvent.change(screen.getByTestId('monaco-editor'), { target: { value: '__undefined__' } });
    expect(onChange).toHaveBeenCalledWith('');
  });
});
