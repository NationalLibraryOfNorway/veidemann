import {Component, ErrorHandler, provideZonelessChangeDetection} from '@angular/core';
import {ComponentFixture, TestBed} from '@angular/core/testing';
import {FormControl, ReactiveFormsModule} from '@angular/forms';
import {By} from '@angular/platform-browser';
import {MonacoEditorComponent, MonacoEditorLanguage, MonacoEditorOptions, MonacoEditorTheme} from './monaco-editor.component';
import {createMonacoEditorTestingHarness, MonacoEditorTestingHarness} from './monaco-editor.spec-helpers';

@Component({
  imports: [MonacoEditorComponent, ReactiveFormsModule],
  template: `
    <app-monaco-editor
      [formControl]="control"
      [language]="language"
      [options]="options"
      [theme]="theme"
      (initialized)="onInitialized($event)" />
  `,
  standalone: true,
})
class TestHostComponent {
  readonly control = new FormControl('const answer = 42;');
  language: MonacoEditorLanguage = 'javascript';
  options: MonacoEditorOptions = {automaticLayout: true, roundedSelection: true};
  theme: MonacoEditorTheme = 'vs';
  readonly onInitialized = vi.fn();
}

describe('MonacoEditorComponent', () => {
  let fixture: ComponentFixture<TestHostComponent>;
  let host: TestHostComponent;
  let harness: MonacoEditorTestingHarness;
  let errorHandler: {handleError: ReturnType<typeof vi.fn>};

  beforeEach(async () => {
    harness = createMonacoEditorTestingHarness();
    errorHandler = {handleError: vi.fn()};
    await TestBed.configureTestingModule({
      imports: [TestHostComponent],
      providers: [
        provideZonelessChangeDetection(),
        harness.provider,
        {provide: ErrorHandler, useValue: errorHandler},
      ],
    }).compileComponents();
    fixture = TestBed.createComponent(TestHostComponent);
    host = fixture.componentInstance;
    fixture.detectChanges();
    await fixture.whenStable();
  });

  it('creates a JavaScript editor with the form value and merged defaults', () => {
    expect(harness.createModel).toHaveBeenCalledWith('const answer = 42;', 'javascript');
    expect(harness.createEditor).toHaveBeenCalledWith(expect.any(HTMLElement), expect.objectContaining({
      automaticLayout: true,
      readOnly: false,
      roundedSelection: true,
      scrollBeyondLastLine: false,
    }));
    expect(host.onInitialized).toHaveBeenCalledOnce();
  });

  it('propagates user edits and blur through the form control', () => {
    harness.emitContent('print("updated")');
    expect(host.control.value).toBe('print("updated")');

    harness.emitBlur();
    expect(host.control.touched).toBe(true);
  });

  it('does not echo programmatic form writes back through the control', () => {
    const valueChanges = vi.fn();
    host.control.valueChanges.subscribe(valueChanges);

    host.control.setValue('const updated = true;');

    expect(valueChanges).toHaveBeenCalledOnce();
  });

  it('maps the disabled state to Monaco read-only mode', () => {
    host.control.disable();
    expect(harness.updateOptions).toHaveBeenLastCalledWith({readOnly: true});

    host.control.enable();
    expect(harness.updateOptions).toHaveBeenLastCalledWith({readOnly: false});
  });

  it('updates language, theme, and options without recreating the editor', async () => {
    host.language = 'python';
    host.theme = 'vs-dark';
    host.options = {automaticLayout: false, minimap: {enabled: false}};
    fixture.detectChanges();
    await fixture.whenStable();

    expect(harness.setModelLanguage).toHaveBeenLastCalledWith(expect.anything(), 'python');
    expect(harness.setTheme).toHaveBeenLastCalledWith('vs-dark');
    expect(harness.updateOptions).toHaveBeenLastCalledWith(expect.objectContaining({
      automaticLayout: false,
      minimap: {enabled: false},
      readOnly: false,
      scrollBeyondLastLine: false,
    }));
    expect(harness.createEditor).toHaveBeenCalledOnce();
  });

  it('disposes listeners, editor, and model with the component', () => {
    fixture.destroy();

    expect(harness.contentListenerDispose).toHaveBeenCalledOnce();
    expect(harness.blurListenerDispose).toHaveBeenCalledOnce();
    expect(harness.editorDispose).toHaveBeenCalledOnce();
    expect(harness.modelDispose).toHaveBeenCalledOnce();
  });

  it('reports loader failures through Angular error handling', async () => {
    const loadError = new Error('Monaco failed to load');
    harness.loader.load.mockRejectedValueOnce(loadError);
    const editor = fixture.debugElement.query(By.directive(MonacoEditorComponent));
    editor.componentInstance.ngOnDestroy();

    const secondFixture = TestBed.createComponent(TestHostComponent);
    secondFixture.detectChanges();
    await secondFixture.whenStable();

    expect(errorHandler.handleError).toHaveBeenCalledWith(loadError);
    secondFixture.destroy();
  });
});
