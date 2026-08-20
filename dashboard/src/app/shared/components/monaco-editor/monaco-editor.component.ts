import {
  AfterViewInit,
  ChangeDetectionStrategy,
  ChangeDetectorRef,
  Component,
  ElementRef,
  ErrorHandler,
  OnDestroy,
  effect,
  inject,
  input,
  output,
  viewChild,
} from '@angular/core';
import {ControlValueAccessor, NG_VALUE_ACCESSOR} from '@angular/forms';
import type {editor, IDisposable} from 'monaco-editor/editor';
import {MonacoApi, MonacoEditorLoader} from './monaco-editor.loader';

export type MonacoEditorLanguage = 'javascript' | 'python';
export type MonacoEditorTheme = editor.BuiltinTheme;
export type MonacoEditorOptions = Omit<
  editor.IStandaloneEditorConstructionOptions,
  'language' | 'model' | 'readOnly' | 'theme' | 'value'
>;

const DEFAULT_OPTIONS: MonacoEditorOptions = {
  scrollBeyondLastLine: false,
};

@Component({
  selector: 'app-monaco-editor',
  template: '<div #editorContainer class="editor-container"></div>',
  styles: `
    :host {
      display: block;
      height: 200px;
      min-width: 0;
    }

    .editor-container {
      width: 100%;
      height: 100%;
    }
  `,
  providers: [{provide: NG_VALUE_ACCESSOR, useExisting: MonacoEditorComponent, multi: true}],
  changeDetection: ChangeDetectionStrategy.OnPush,
  standalone: true,
})
export class MonacoEditorComponent implements AfterViewInit, OnDestroy, ControlValueAccessor {
  private readonly changeDetectorRef = inject(ChangeDetectorRef);
  private readonly errorHandler = inject(ErrorHandler);
  private readonly loader = inject(MonacoEditorLoader);
  private readonly editorContainer = viewChild.required<ElementRef<HTMLElement>>('editorContainer');

  readonly options = input<MonacoEditorOptions>({});
  readonly language = input<MonacoEditorLanguage>('javascript');
  readonly theme = input<MonacoEditorTheme>('vs');
  readonly initialized = output<editor.IStandaloneCodeEditor>();

  private monaco?: MonacoApi;
  private editor?: editor.IStandaloneCodeEditor;
  private model?: editor.ITextModel;
  private readonly listeners: IDisposable[] = [];
  private value = '';
  private disabled = false;
  private writingValue = false;
  private destroyed = false;
  private onChange: (value: string) => void = () => undefined;
  private onTouched: () => void = () => undefined;

  constructor() {
    effect(() => {
      const options = this.options();
      const language = this.language();
      const theme = this.theme();
      if (this.monaco && this.editor && this.model) {
        this.applyConfiguration(options, language, theme);
      }
    });
  }

  ngAfterViewInit(): void {
    void this.initialize();
  }

  writeValue(value: string | null | undefined): void {
    this.value = value ?? '';
    if (this.model && this.model.getValue() !== this.value) {
      this.writingValue = true;
      try {
        this.model.setValue(this.value);
      } finally {
        this.writingValue = false;
      }
    }
  }

  registerOnChange(fn: (value: string) => void): void {
    this.onChange = fn;
  }

  registerOnTouched(fn: () => void): void {
    this.onTouched = fn;
  }

  setDisabledState(disabled: boolean): void {
    this.disabled = disabled;
    this.editor?.updateOptions({readOnly: disabled});
  }

  ngOnDestroy(): void {
    this.destroyed = true;
    this.disposeEditor();
  }

  private disposeEditor(): void {
    this.listeners.splice(0).forEach(listener => listener.dispose());
    this.editor?.dispose();
    this.editor = undefined;
    this.model?.dispose();
    this.model = undefined;
  }

  private async initialize(): Promise<void> {
    try {
      const monaco = await this.loader.load();
      if (this.destroyed) {
        return;
      }

      this.monaco = monaco;
      this.model = monaco.editor.createModel(this.value, this.language());
      monaco.editor.setTheme(this.theme());
      this.editor = monaco.editor.create(this.editorContainer().nativeElement, {
        ...DEFAULT_OPTIONS,
        ...this.options(),
        model: this.model,
        readOnly: this.disabled,
      });
      this.listeners.push(
        this.editor.onDidChangeModelContent(() => this.handleContentChange()),
        this.editor.onDidBlurEditorWidget(() => this.handleBlur()),
      );
      this.initialized.emit(this.editor);
    } catch (error) {
      if (!this.destroyed) {
        this.disposeEditor();
        this.errorHandler.handleError(error);
      }
    }
  }

  private applyConfiguration(
    options: MonacoEditorOptions,
    language: MonacoEditorLanguage,
    theme: MonacoEditorTheme,
  ): void {
    this.monaco.editor.setTheme(theme);
    this.monaco.editor.setModelLanguage(this.model, language);
    this.editor.updateOptions({...DEFAULT_OPTIONS, ...options, readOnly: this.disabled});
  }

  private handleContentChange(): void {
    if (this.writingValue || !this.editor) {
      return;
    }
    const value = this.editor.getValue();
    this.value = value;
    this.onChange(value);
    this.changeDetectorRef.markForCheck();
  }

  private handleBlur(): void {
    this.onTouched();
    this.changeDetectorRef.markForCheck();
  }
}
