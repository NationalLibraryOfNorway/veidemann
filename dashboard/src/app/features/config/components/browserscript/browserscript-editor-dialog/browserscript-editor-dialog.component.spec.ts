import {ComponentFixture, TestBed} from '@angular/core/testing';
import {MAT_DIALOG_DATA, MatDialogRef} from '@angular/material/dialog';
import {MonacoEditorModule} from 'ngx-monaco-editor-v2';
import {vi} from 'vitest';

import {provideCoreTesting} from '../../../../../core/core.testing.module';
import {
  BrowserScriptEditorDialogComponent,
  BrowserScriptEditorDialogData
} from './browserscript-editor-dialog.component';

describe('BrowserScriptEditorDialogComponent', () => {
  let fixture: ComponentFixture<BrowserScriptEditorDialogComponent>;
  const data: BrowserScriptEditorDialogData = {
    name: 'Example BrowserScript',
    script: 'console.log(\'test\')',
    readOnly: false,
    theme: 'vs',
  };
  const dialogRef = {
    close: vi.fn(),
  };

  beforeEach(() => {
    TestBed.configureTestingModule({
      imports: [
        BrowserScriptEditorDialogComponent,
        MonacoEditorModule.forRoot(),
      ],
      providers: [
        ...provideCoreTesting,
        {provide: MAT_DIALOG_DATA, useValue: data},
        {provide: MatDialogRef, useValue: dialogRef},
      ],
    }).compileComponents();

    data.name = 'Example BrowserScript';
    data.script = 'console.log(\'test\')';
    data.readOnly = false;
    data.theme = 'vs';
    dialogRef.close.mockReset();
  });

  function createComponent(): BrowserScriptEditorDialogComponent {
    fixture = TestBed.createComponent(BrowserScriptEditorDialogComponent);
    fixture.detectChanges();
    return fixture.componentInstance;
  }

  it('initializes an editable full-screen Monaco draft', () => {
    const component = createComponent();
    const text = (fixture.nativeElement as HTMLElement).textContent;

    expect(component.scriptControl.value).toBe(data.script);
    expect(component.editorOptions).toEqual(expect.objectContaining({
      theme: 'vs',
      language: 'javascript',
      automaticLayout: true,
      readOnly: false,
    }));
    expect(component.canApply).toBe(false);
    expect(component.canRevert).toBe(false);
    expect(text).toContain('Edit script: Example BrowserScript');
  });

  it('applies a changed Monaco value without relying on the dirty flag', () => {
    const component = createComponent();
    component.scriptControl.setValue('console.log(\'updated\')');
    fixture.detectChanges();
    const applyButton = Array.from<HTMLButtonElement>(fixture.nativeElement.querySelectorAll('button'))
      .find(button => button.textContent.includes('APPLY'));

    expect(component.scriptControl.pristine).toBe(true);
    expect(component.canApply).toBe(true);
    expect(applyButton.disabled).toBe(false);
    component.onApply();

    expect(dialogRef.close).toHaveBeenCalledWith({script: 'console.log(\'updated\')'});
  });

  it('can explicitly apply an empty script', () => {
    const component = createComponent();
    component.scriptControl.setValue('');

    expect(component.canApply).toBe(true);
    component.onApply();

    expect(dialogRef.close).toHaveBeenCalledWith({script: ''});
  });

  it('cancels without returning a script result', () => {
    createComponent();
    const cancelButton = Array.from<HTMLButtonElement>(fixture.nativeElement.querySelectorAll('button'))
      .find(button => button.textContent.includes('CANCEL'));

    cancelButton.click();

    expect(dialogRef.close).toHaveBeenCalledWith();
  });

  it('reverts to the value from when the dialog opened', () => {
    const component = createComponent();
    component.scriptControl.setValue('console.log(\'discarded\')');

    expect(component.canRevert).toBe(true);
    component.onRevert();

    expect(component.scriptControl.value).toBe(data.script);
    expect(component.canApply).toBe(false);
    expect(component.canRevert).toBe(false);
  });

  it('renders a read-only viewer with no Apply action', () => {
    data.readOnly = true;
    const component = createComponent();
    const text = (fixture.nativeElement as HTMLElement).textContent;

    expect(component.editorOptions.readOnly).toBe(true);
    expect(component.canApply).toBe(false);
    expect(text).toContain('View script: Example BrowserScript');
    expect(text).toContain('CLOSE');
    expect(text).not.toContain('APPLY');
    expect(text).not.toContain('REVERT');
  });
});
