import {Provider} from '@angular/core';
import {vi} from 'vitest';
import {MonacoApi, MonacoEditorLoader} from './monaco-editor.loader';

export interface MonacoEditorTestingHarness {
  readonly provider: Provider;
  readonly loader: {load: ReturnType<typeof vi.fn>};
  readonly createEditor: ReturnType<typeof vi.fn>;
  readonly createModel: ReturnType<typeof vi.fn>;
  readonly setModelLanguage: ReturnType<typeof vi.fn>;
  readonly setTheme: ReturnType<typeof vi.fn>;
  readonly updateOptions: ReturnType<typeof vi.fn>;
  readonly editorDispose: ReturnType<typeof vi.fn>;
  readonly modelDispose: ReturnType<typeof vi.fn>;
  readonly contentListenerDispose: ReturnType<typeof vi.fn>;
  readonly blurListenerDispose: ReturnType<typeof vi.fn>;
  emitContent(value: string): void;
  emitBlur(): void;
}

export function createMonacoEditorTestingHarness(): MonacoEditorTestingHarness {
  let value = '';
  let contentListener = () => undefined;
  let blurListener = () => undefined;

  const modelDispose = vi.fn();
  const contentListenerDispose = vi.fn();
  const blurListenerDispose = vi.fn();
  const updateOptions = vi.fn();
  const editorDispose = vi.fn();

  const model = {
    getValue: vi.fn(() => value),
    setValue: vi.fn((newValue: string) => {
      value = newValue;
      contentListener();
    }),
    dispose: modelDispose,
  };
  const editor = {
    getValue: vi.fn(() => value),
    updateOptions,
    dispose: editorDispose,
    onDidChangeModelContent: vi.fn((listener: () => void) => {
      contentListener = listener;
      return {dispose: contentListenerDispose};
    }),
    onDidBlurEditorWidget: vi.fn((listener: () => void) => {
      blurListener = listener;
      return {dispose: blurListenerDispose};
    }),
  };
  const createModel = vi.fn((initialValue: string) => {
    value = initialValue;
    return model;
  });
  const createEditor = vi.fn(() => editor);
  const setModelLanguage = vi.fn();
  const setTheme = vi.fn();
  const api = {
    editor: {
      create: createEditor,
      createModel,
      setModelLanguage,
      setTheme,
    },
  } as unknown as MonacoApi;
  const loader = {load: vi.fn(() => Promise.resolve(api))};

  return {
    provider: {provide: MonacoEditorLoader, useValue: loader},
    loader,
    createEditor,
    createModel,
    setModelLanguage,
    setTheme,
    updateOptions,
    editorDispose,
    modelDispose,
    contentListenerDispose,
    blurListenerDispose,
    emitContent: (newValue: string) => {
      value = newValue;
      contentListener();
    },
    emitBlur: () => blurListener(),
  };
}
