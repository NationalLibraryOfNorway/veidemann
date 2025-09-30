import {ComponentFixture, TestBed} from '@angular/core/testing';

import {MetaComponent} from './meta.component';
import {LabelComponent} from '../label/label.component';
import {provideCoreTesting} from '../../../../core/core.testing.module';
import {ActivatedRoute} from '@angular/router';
import {of} from 'rxjs';

describe('MetaComponent', () => {
  let component: MetaComponent;
  let fixture: ComponentFixture<MetaComponent>;

  beforeEach(() => {
    TestBed.configureTestingModule({
      imports: [
        LabelComponent
      ],
      providers: [
        ...provideCoreTesting,
        { provide: ActivatedRoute, useValue: { snapshot: {}, params: of({}), queryParams: of({}) } }
      ]
    })
      .compileComponents();
  });

  beforeEach(async () => {
    fixture = TestBed.createComponent(MetaComponent);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  it('form field name is valid if it contains 2 or more characters', async () => {
    const name = component.form.controls.name;
    name.setValue('a');
    await fixture.whenStable();
    expect(name.status === 'INVALID').toBeTruthy();
    name.setValue('ab');
    await fixture.whenStable();
    expect(name.status === 'VALID').toBeTruthy();
  });

  it('form is not valid if required fields are missing', async () => {
    const name = component.form.controls.name;
    name.markAsTouched();
    await fixture.whenStable();
    expect(component.form.status === 'INVALID').toBeTruthy();
  });

  it('form is valid if required fields are set', async () => {
    const name = component.form.controls.name;
    name.setValue('Test');
    await fixture.whenStable();
    expect(component.form.status === 'VALID').toBeTruthy();
  });
});
