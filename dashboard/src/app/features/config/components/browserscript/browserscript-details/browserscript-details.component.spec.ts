import {ComponentFixture, TestBed} from '@angular/core/testing';
import {By} from '@angular/platform-browser';
import {BrowserScriptDetailsComponent} from './browserscript-details.component';
import {SimpleChange} from '@angular/core';
import {
  Annotation,
  BrowserScript,
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
import {EditorComponent, MonacoEditorModule} from 'ngx-monaco-editor-v2';
import {AuthService} from '../../../../../core';
import {provideCoreTesting} from '../../../../../core/core.testing.module';
import {vi} from 'vitest';
import {MatDialog} from '@angular/material/dialog';
import {Subject} from 'rxjs';
import {
  BrowserScriptEditorDialogComponent,
  BrowserScriptEditorDialogResult
} from '../browserscript-editor-dialog/browserscript-editor-dialog.component';


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
  let dialogClosed: Subject<BrowserScriptEditorDialogResult | undefined>;

  const dialog = {
    open: vi.fn(),
  };
  const authService = {
    canUpdate: vi.fn(() => true),
    canDelete: vi.fn(() => true),
  };

  // Async beforeEach needed when using external template
  beforeEach(() => {
    TestBed.configureTestingModule({
      imports: [
        BrowserScriptDetailsComponent,
        MonacoEditorModule.forRoot()
      ],
      providers: [
        ...provideCoreTesting,
        {provide: AuthService, useValue: authService},
        {provide: MatDialog, useValue: dialog},
      ]
    })
      .compileComponents();
  });

  beforeEach(async () => {
    dialogClosed = new Subject<BrowserScriptEditorDialogResult | undefined>();
    dialog.open.mockReset();
    dialog.open.mockReturnValue({afterClosed: () => dialogClosed.asObservable()});
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

    expect(urlRegexp.parentElement).toBe(scriptColumn);
    expect(editorSection.parentElement).toBe(scriptColumn);
    expect(Array.from(scriptColumn.children).indexOf(editorSection))
      .toBeGreaterThan(Array.from(scriptColumn.children).indexOf(urlRegexp));
    const editor = editorSection.querySelector('ngx-monaco-editor.editor-resizable') as HTMLElement;
    expect(editor).not.toBeNull();
    expect(getComputedStyle(editor).resize).toBe('vertical');
    expect(component.editorOptions.automaticLayout).toBe(true);
  });

  it('opens the current script in a full-screen editor dialog', () => {
    component.onOpenFullscreenEditor();

    expect(dialog.open).toHaveBeenCalledWith(
      BrowserScriptEditorDialogComponent,
      expect.objectContaining({
        data: {
          name: 'Example BrowserScript',
          script: 'console.log(\'test\')',
          readOnly: false,
          theme: 'vs',
        },
      })
    );
  });

  it('applies a full-screen edit and marks the form dirty', () => {
    component.onOpenFullscreenEditor();

    dialogClosed.next({script: 'console.log(\'updated\')'});
    dialogClosed.complete();

    expect(component.script.value).toBe('console.log(\'updated\')');
    expect(component.script.dirty).toBe(true);
    expect(component.form.dirty).toBe(true);
  });

  it('applies an intentionally empty script', () => {
    component.onOpenFullscreenEditor();

    dialogClosed.next({script: ''});
    dialogClosed.complete();

    expect(component.script.value).toBe('');
    expect(component.form.dirty).toBe(true);
  });

  it('discards a canceled full-screen edit', () => {
    component.onOpenFullscreenEditor();

    dialogClosed.next(undefined);
    dialogClosed.complete();

    expect(component.script.value).toBe('console.log(\'test\')');
    expect(component.form.pristine).toBe(true);
  });

  it('opens the full-screen editor read-only without update permission', () => {
    authService.canUpdate.mockReturnValue(false);

    component.onOpenFullscreenEditor();

    expect(dialog.open.mock.calls[0][1].data.readOnly).toBe(true);
  });

  it('uses the light Monaco theme when the dashboard is in light mode', () => {
    expect(component.editorOptions.theme).toBe('vs');
  });

  it('updates the Monaco theme when the preferred color scheme changes', async () => {
    const editorComponent = fixture.debugElement.query(By.directive(EditorComponent)).componentInstance as EditorComponent;
    const setTheme = vi.spyOn(editorComponent, 'setTheme').mockImplementation(() => undefined);
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

    component.initEditor();
    fixture.detectChanges();
    await fixture.whenStable();
    setTheme.mockClear();

    changeListener({matches: true} as MediaQueryListEvent);
    fixture.detectChanges();
    await fixture.whenStable();

    expect(setTheme).toHaveBeenCalledWith('vs-dark');

    setTheme.mockClear();
    changeListener({matches: false} as MediaQueryListEvent);
    fixture.detectChanges();
    await fixture.whenStable();

    expect(setTheme).toHaveBeenCalledWith('vs');

    fixture.destroy();
    expect(mediaQuery.removeEventListener).toHaveBeenCalledWith('change', changeListener);
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
