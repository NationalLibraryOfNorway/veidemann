import {ComponentFixture, TestBed} from '@angular/core/testing';

import {provideCoreTesting} from '../../../../../core/core.testing.module';
import {SubCollection, SubCollectionType} from '../../../../../shared/models';
import {SubcollectionChipsComponent} from './subcollection-chips.component';

describe('SubcollectionChipsComponent', () => {
  let fixture: ComponentFixture<SubcollectionChipsComponent>;
  let component: SubcollectionChipsComponent;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [SubcollectionChipsComponent],
      providers: [...provideCoreTesting],
    }).compileComponents();
    fixture = TestBed.createComponent(SubcollectionChipsComponent);
    component = fixture.componentInstance;
    component.types = [SubCollectionType.UNDEFINED, SubCollectionType.SCREENSHOT, SubCollectionType.DNS];
    component.writeValue([new SubCollection({name: 'screenshots', type: SubCollectionType.SCREENSHOT})]);
    fixture.detectChanges();
  });

  it('renders inline fields and hides the add action for an existing type', () => {
    expect(fixture.nativeElement.querySelectorAll('[data-testid="subcollectionRow"]').length).toBe(1);
    expect(component.nameControl(0).value).toBe('screenshots');
    expect(component.typeControl(0).value).toBe(SubCollectionType.SCREENSHOT);
    expect(component.typeControl(0).disabled).toBe(true);
    expect(fixture.nativeElement.querySelector('[data-testid="addScreenshotSubcollectionButton"]')).toBeNull();
    expect(fixture.nativeElement.querySelector('[data-testid="addDnsSubcollectionButton"]')).not.toBeNull();
  });

  it('adds and removes one subcollection of each type', () => {
    const changes: SubCollection[][] = [];
    component.registerOnChange(value => changes.push(value));

    component.add(SubCollectionType.DNS);
    fixture.detectChanges();
    expect(changes.at(-1)?.map(item => item.type)).toEqual([
      SubCollectionType.SCREENSHOT,
      SubCollectionType.DNS,
    ]);
    expect(fixture.nativeElement.querySelector('[data-testid="addDnsSubcollectionButton"]')).toBeNull();

    component.add(SubCollectionType.DNS);
    expect(component.itemControls.length).toBe(2);

    component.remove(1);
    fixture.detectChanges();
    expect(component.itemControls.length).toBe(1);
    expect(fixture.nativeElement.querySelector('[data-testid="addDnsSubcollectionButton"]')).not.toBeNull();
  });

  it('supports keyboard add actions', () => {
    const addDns = fixture.nativeElement.querySelector('[data-testid="addDnsSubcollectionButton"]');
    addDns.dispatchEvent(new KeyboardEvent('keydown', {key: ' ', bubbles: true}));
    fixture.detectChanges();
    expect(component.hasType(SubCollectionType.DNS)).toBe(true);
  });

  it('reports invalid names and duplicate types to the parent form', () => {
    component.nameControl(0).setValue('x');
    expect(component.validate()).toEqual({invalidSubcollections: true});

    component.writeValue([
      new SubCollection({name: 'screenshots', type: SubCollectionType.SCREENSHOT}),
      new SubCollection({name: 'other_screenshots', type: SubCollectionType.SCREENSHOT}),
    ]);
    fixture.detectChanges();
    expect(component.validate()).toEqual({duplicateSubcollectionTypes: true});
    expect(fixture.nativeElement.querySelector('[data-testid="duplicateSubcollectionTypeError"]')).not.toBeNull();
  });

  it('hides editing controls and disables fields in read-only mode', () => {
    component.setDisabledState(true);
    fixture.detectChanges();
    expect(fixture.nativeElement.querySelector('[data-testid="addDnsSubcollectionButton"]')).toBeNull();
    expect(fixture.nativeElement.querySelector('[data-testid="removeSubcollectionButton"]')).toBeNull();
    expect(component.nameControl(0).disabled).toBe(true);
  });
});
