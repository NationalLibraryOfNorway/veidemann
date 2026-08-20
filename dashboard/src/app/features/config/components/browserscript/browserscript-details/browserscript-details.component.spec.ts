import {ComponentFixture, TestBed} from '@angular/core/testing';
import {By} from '@angular/platform-browser';
import {BrowserScriptDetailsComponent} from './browserscript-details.component';
import {SimpleChange} from '@angular/core';
import {
  Annotation,
  BrowserScript,
  BrowserScriptType,
  browserScriptTypes,
  ConfigObject,
  Kind,
  Label,
  Meta
} from '../../../../../shared/models';
import {HarnessLoader} from '@angular/cdk/testing';
import {MatButtonHarness} from '@angular/material/button/testing';
import {MatSelectHarness} from '@angular/material/select/testing';
import {TestbedHarnessEnvironment} from '@angular/cdk/testing/testbed';
import {AuthService} from '../../../../../core';
import {provideCoreTesting} from '../../../../../core/core.testing.module';
import {vi} from 'vitest';
import {MonacoEditorComponent} from '../../../../../shared/components';
import {
  createMonacoEditorTestingHarness,
  MonacoEditorTestingHarness
} from '../../../../../shared/components/monaco-editor/monaco-editor.spec-helpers';


const exampleBrowserScript: ConfigObject = {
  id: 'configObject_id',
  apiVersion: 'v1',
  kind: Kind.BROWSERSCRIPT,
  meta: new Meta({
    name: 'Example BrowserScript',
    createdBy: 'test',
    created: '01.01.1970',
    lastModified: '01.01.2021',
    lastModifiedBy: 'test',
    description: 'This is an example BrowserScript',
    labelList: [new Label({key: 'test', value: 'label'})],
    annotationList: [new Annotation({key: 'test', value: 'annotation'})]
  }),
  browserScript: new BrowserScript({
    script: 'console.log(\'test\')',
    urlRegexpList: [],
    browserScriptType: null
  })
};

describe('BrowserScriptDetailsComponent', () => {
  let component: BrowserScriptDetailsComponent;
  let fixture: ComponentFixture<BrowserScriptDetailsComponent>;
  let loader: HarnessLoader;

  let saveButton: MatButtonHarness;
  let updateButton: MatButtonHarness;

  let scriptTypeSelect: MatSelectHarness;
  let monaco: MonacoEditorTestingHarness;
  const authService = {
    canUpdate: vi.fn(() => true),
    canDelete: vi.fn(() => true),
  };

  // Async beforeEach needed when using external template
  beforeEach(() => {
    monaco = createMonacoEditorTestingHarness();
    TestBed.configureTestingModule({
      imports: [
        BrowserScriptDetailsComponent,
      ],
      providers: [
        ...provideCoreTesting,
        monaco.provider,
        {provide: AuthService, useValue: authService},
      ]
    })
      .compileComponents();
  });

  beforeEach(async () => {
    authService.canUpdate.mockReturnValue(true);

    fixture = TestBed.createComponent(BrowserScriptDetailsComponent);
    loader = TestbedHarnessEnvironment.loader(fixture);
    component = fixture.componentInstance;
    component.configObject = new ConfigObject(exampleBrowserScript);
    component.browserScriptTypes = browserScriptTypes;
    component.ngOnChanges({
      configObject: new SimpleChange(null, component.configObject, null)
    });
    await fixture.whenStable();
    // await fixture.whenStable();

    scriptTypeSelect = await loader.getHarness<MatSelectHarness>(MatSelectHarness);
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  it('offers a copy button for the saved ID field', () => {
    expect(fixture.nativeElement.querySelector('button[aria-label="Copy ID"]')).not.toBeNull();
  });

  it('places the editor under URL regexp in the right-hand column', () => {
    const scriptColumn = fixture.nativeElement.querySelector('.script-column') as HTMLElement;
    const urlRegexp = fixture.nativeElement.querySelector('.url-regexp-field') as HTMLElement;
    const editorSection = fixture.nativeElement.querySelector('.editor-section') as HTMLElement;
    const editorField = editorSection.querySelector('.editor-field') as HTMLElement;

    expect(urlRegexp.parentElement).toBe(scriptColumn);
    expect(editorSection.parentElement).toBe(scriptColumn);
    expect(Array.from(scriptColumn.children).indexOf(editorSection))
      .toBeGreaterThan(Array.from(scriptColumn.children).indexOf(urlRegexp));
    const editor = editorSection.querySelector('app-monaco-editor.editor-resizable') as HTMLElement;
    expect(editor).not.toBeNull();
    expect(editor.parentElement).toBe(editorField);
    expect(getComputedStyle(editorField).borderBlockEndStyle).toBe('solid');
    expect(getComputedStyle(editorField).borderTopLeftRadius).not.toBe('0px');
    expect(getComputedStyle(editor).resize).toBe('vertical');
    expect(component.editorOptions.automaticLayout).toBe(true);
  });

  it('places an icon-only edit-script action right aligned with update and revert', () => {
    const actionRow = fixture.nativeElement.querySelector('.config-form-actions') as HTMLElement;
    const editButton = actionRow.querySelector(
      '[data-testid="toggleScriptExpandedButton"]'
    ) as HTMLButtonElement;

    expect(editButton.textContent.trim()).toBe('open_in_full');
    expect(editButton.getAttribute('aria-label')).toBe('Edit script');
    expect(editButton.getAttribute('aria-pressed')).toBe('false');
    expect(editButton.previousElementSibling.classList).toContain('flex-fill');
  });

  it('grows the script editor to the bottom of its detail-grid column', () => {
    const detailsGrid = fixture.nativeElement.querySelector('.details-grid') as HTMLElement;
    const editorSection = fixture.nativeElement.querySelector('.editor-section') as HTMLElement;
    const editorField = fixture.nativeElement.querySelector('.editor-field') as HTMLElement;
    const editor = editorField.querySelector('app-monaco-editor') as HTMLElement;

    expect(getComputedStyle(detailsGrid).alignItems).toBe('stretch');
    expect(editorSection.classList).toContain('flex-fill');
    expect(editorField.classList).toContain('flex-fill');
    expect(getComputedStyle(editor).height).toBe('100%');
    expect(getComputedStyle(editorField).minHeight).toBe('280px');
  });

  it('expands the script as the sole form field while keeping form actions visible', () => {
    component.toggleScriptExpanded();
    fixture.detectChanges();

    const detailsGrid = fixture.nativeElement.querySelector('.details-grid') as HTMLElement;
    const editButton = fixture.nativeElement.querySelector(
      '[data-testid="toggleScriptExpandedButton"]'
    ) as HTMLButtonElement;

    expect(detailsGrid.classList).toContain('script-expanded');
    expect(fixture.nativeElement.querySelector('app-meta')).toBeNull();
    expect(fixture.nativeElement.querySelector('.url-regexp-field')).toBeNull();
    expect(fixture.nativeElement.querySelector('mat-select')).toBeNull();
    expect(fixture.nativeElement.querySelector('.editor-field')).not.toBeNull();
    const editorField = fixture.nativeElement.querySelector('.editor-field') as HTMLElement;
    const formActions = fixture.nativeElement.querySelector('.config-form-actions') as HTMLElement;
    expect(formActions).not.toBeNull();
    expect(detailsGrid.compareDocumentPosition(formActions) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
    expect(getComputedStyle(detailsGrid).minHeight).toBe('400px');
    expect(getComputedStyle(editorField).minHeight).toBe('0px');
    expect(editButton.textContent.trim()).toBe('close_fullscreen');
    expect(editButton.getAttribute('aria-label')).toBe('Show full form');
    expect(editButton.getAttribute('aria-pressed')).toBe('true');
  });

  it('edits and updates the form script directly from expanded mode', () => {
    const update = vi.spyOn(component.update, 'emit');
    component.toggleScriptExpanded();
    component.script.setValue('console.log(\'updated\')');
    component.script.markAsDirty();
    component.form.markAsDirty();
    component.onUpdate();

    expect(update).toHaveBeenCalledWith(expect.objectContaining({
      browserScript: expect.objectContaining({script: 'console.log(\'updated\')'}),
    }));
  });

  it('restores the other fields when leaving expanded mode', () => {
    component.toggleScriptExpanded();
    component.toggleScriptExpanded();
    fixture.detectChanges();

    expect(fixture.nativeElement.querySelector('app-meta')).not.toBeNull();
    expect(fixture.nativeElement.querySelector('.url-regexp-field')).not.toBeNull();
    expect(fixture.nativeElement.querySelector('mat-select')).not.toBeNull();
  });

  it('uses the light Monaco theme when the dashboard is in light mode', () => {
    expect(component.editorTheme()).toBe('vs');
  });

  it('updates the Monaco theme when the preferred color scheme changes', async () => {
    const matchMedia = vi.mocked(window.matchMedia);
    let queryIndex = -1;
    matchMedia.mock.calls.forEach(([query], index) => {
      if (query === '(prefers-color-scheme: dark)') {
        queryIndex = index;
      }
    });
    const mediaQuery = matchMedia.mock.results[queryIndex].value;
    const changeListener = vi.mocked(mediaQuery.addEventListener).mock.calls.find(
      ([type]) => type === 'change'
    )[1] as EventListener;

    monaco.setTheme.mockClear();

    changeListener({matches: true} as MediaQueryListEvent);
    fixture.detectChanges();
    await fixture.whenStable();

    expect(component.editorTheme()).toBe('vs-dark');
    expect(monaco.setTheme).toHaveBeenCalledWith('vs-dark');

    monaco.setTheme.mockClear();
    changeListener({matches: false} as MediaQueryListEvent);
    fixture.detectChanges();
    await fixture.whenStable();

    expect(component.editorTheme()).toBe('vs');
    expect(monaco.setTheme).toHaveBeenCalledWith('vs');

    fixture.destroy();
    expect(mediaQuery.removeEventListener).toHaveBeenCalledWith('change', changeListener);
  });

  it('uses Python highlighting for Starlark scope-check scripts', async () => {
    component.form.get('browserScriptType').setValue(BrowserScriptType.SCOPE_CHECK);
    fixture.detectChanges();
    await fixture.whenStable();

    const editor = fixture.debugElement.query(By.directive(MonacoEditorComponent))
      .componentInstance as MonacoEditorComponent;
    expect(component.editorLanguage).toBe('python');
    expect(editor.language()).toBe('python');
    expect(monaco.setModelLanguage).toHaveBeenLastCalledWith(expect.anything(), 'python');
  });

  it('uses JavaScript highlighting for browser-executed scripts', async () => {
    component.form.get('browserScriptType').setValue(BrowserScriptType.SCOPE_CHECK);
    fixture.detectChanges();
    await fixture.whenStable();
    monaco.setModelLanguage.mockClear();

    component.form.get('browserScriptType').setValue(BrowserScriptType.ON_LOAD);
    fixture.detectChanges();
    await fixture.whenStable();

    const editor = fixture.debugElement.query(By.directive(MonacoEditorComponent))
      .componentInstance as MonacoEditorComponent;
    expect(component.editorLanguage).toBe('javascript');
    expect(editor.language()).toBe('javascript');
    expect(monaco.setModelLanguage).toHaveBeenLastCalledWith(expect.anything(), 'javascript');
  });

  describe('Creating a new browserscript', () => {
    beforeEach(async () => {
      component.configObject.id = '';
      component.ngOnChanges({
        configObject: new SimpleChange(null, component.configObject, null)
      });
      await fixture.whenStable();
      saveButton = await loader.getHarness<MatButtonHarness>(MatButtonHarness.with({text: 'SAVE'}));
    });

    it('show save button when creating a new config if form is valid', async () => {
      expect(await saveButton.isDisabled()).toBeFalsy();
      expect(component.canSave).toBeTruthy();
    });

  });

  describe('Updating a browserscript', () => {
    beforeEach(async () => {
      await fixture.whenStable();
      updateButton = await loader.getHarness<MatButtonHarness>(MatButtonHarness.with({text: 'UPDATE'}));
    });

    it('update button should be active if form is updated and valid', async () => {
      expect(await updateButton.isDisabled()).toBeTruthy();
      expect(component.canUpdate).toBeFalsy();
      await scriptTypeSelect.open();
      const scriptTypeOptions = await scriptTypeSelect.getOptions({text: 'ON_LOAD'});
      await scriptTypeOptions[0].click();

      await fixture.whenStable();

      expect(await updateButton.isDisabled()).toBeFalsy();
      expect(component.canUpdate).toBeTruthy();
    });

    it('script type dropdown should be filled with all script type options', async () => {
      await scriptTypeSelect.open();
      const scriptTypeOptions = await scriptTypeSelect.getOptions();
      await scriptTypeSelect.close();
      expect(await Promise.all(scriptTypeOptions.map(option => option.getText()))).toEqual([
        'UNDEFINED',
        'EXTRACT_OUTLINKS',
        'ON_LOAD',
        'ON_NEW_DOCUMENT',
        'SCOPE_CHECK',
      ]);
    });
  });
});
